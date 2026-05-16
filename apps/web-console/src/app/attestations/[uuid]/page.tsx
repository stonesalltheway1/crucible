import { PageHeader } from "@/components/page-header";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { HashPill } from "@/components/hash-pill";
import { formatRelative } from "@/lib/utils";
import { mockAttestation } from "@/lib/mocks";
import { VerifyButton } from "./_verify-button";
import { ShareButton } from "./_share-button";
import { CertificateRow } from "./_cert-row";
import { ShieldCheck, FileSignature, Link as LinkIcon } from "lucide-react";

export default async function AttestationDetailPage({
  params,
}: {
  params: Promise<{ uuid: string }>;
}) {
  const { uuid } = await params;
  const a = mockAttestation(decodeURIComponent(uuid));

  return (
    <>
      <PageHeader
        title="Attestation"
        description={
          <span className="font-mono text-xs">
            {a.predicate_type} · subject={a.subject.name}
          </span>
        }
        actions={
          <>
            <Badge tone={a.validation === "valid" ? "ok" : a.validation === "invalid" ? "alert" : "warn"}>
              {a.validation}
            </Badge>
            {a.self_hosted && <Badge tone="info">self-hosted rekor</Badge>}
          </>
        }
      />

      <div className="grid grid-cols-[1fr_320px] gap-4">
        <div className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <FileSignature className="h-3.5 w-3.5" /> Statement
              </CardTitle>
              <CardDescription>
                in-toto v1 statement: predicate type + subject digest + predicate body. Signed with the agent's
                Sigstore-keyless OIDC identity.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              <Row label="rekor uuid" value={<HashPill value={a.rekor_uuid} />} />
              <Row label="predicate type" value={<Badge tone="info">{a.predicate_type}</Badge>} />
              <Row label="subject name" value={<span className="font-mono text-xs">{a.subject.name}</span>} />
              {Object.entries(a.subject.digest).map(([alg, digest]) => (
                <Row key={alg} label={alg} value={<HashPill value={digest} />} />
              ))}
              <Row label="signed by" value={<span className="font-mono text-xs">{a.signed_by_oidc}</span>} />
              <Row label="signed at" value={<span className="text-xs">{formatRelative(a.signed_at)}</span>} />
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Predicate body</CardTitle>
              <CardDescription>The canonical JSON; the byte-string here is what the DSSE PAE wrapped.</CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="hash-block max-h-96 overflow-auto whitespace-pre-wrap text-[11px]">
{JSON.stringify(a.predicate, null, 2)}
              </pre>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Inclusion proof</CardTitle>
              <CardDescription>
                Merkle proof against the transparency log's signed tree head. Verifiable without contacting the relay.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <pre className="hash-block max-h-72 overflow-auto whitespace-pre-wrap text-[11px]">
{JSON.stringify(a.rekor_inclusion_proof, null, 2)}
              </pre>
            </CardContent>
          </Card>

          {a.cert_chain && a.cert_chain.length > 0 && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <ShieldCheck className="h-3.5 w-3.5" /> Certificate chain
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                {a.cert_chain.map((c, i) => (
                  <CertificateRow key={i} pem={c} />
                ))}
              </CardContent>
            </Card>
          )}
        </div>

        <aside className="space-y-3">
          <Card>
            <CardHeader>
              <CardTitle>Verify</CardTitle>
              <CardDescription>End-to-end against the Sigstore root.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              <VerifyButton rekorUuid={a.rekor_uuid} />
              <p className="text-xs text-muted-foreground">
                Verification re-checks the certificate chain, the Merkle inclusion proof against the latest signed tree
                head, and the subject digest. No backend round-trip beyond the initial attestation fetch.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>Public share</CardTitle>
              <CardDescription>For compliance auditors.</CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              <ShareButton rekorUuid={a.rekor_uuid} />
              <p className="text-xs text-muted-foreground">
                Generates a 30-day signed URL serving the canonical attestation JSON. The link only ever exposes this
                single attestation — no other tenant state.
              </p>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <LinkIcon className="h-3.5 w-3.5" /> Reproduce locally
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 text-xs">
              <code className="block hash-block whitespace-pre-wrap text-[11px]">
{`crucible attestation verify ${a.rekor_uuid}`}
              </code>
            </CardContent>
          </Card>
        </aside>
      </div>
    </>
  );
}

function Row({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-3">
      <span className="font-mono text-[10px] uppercase tracking-wide text-muted-foreground">{label}</span>
      <span className="min-w-0 truncate text-right">{value}</span>
    </div>
  );
}
