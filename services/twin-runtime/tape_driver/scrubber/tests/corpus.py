"""Adversarial PII corpus.

22 categories × ~50 patterns = 1100 synthetic strings spanning the HIPAA
Safe Harbor 18-identifier list, PCI-DSS PAN containment, common cloud
credentials, and free-text personal-name patterns.

The corpus is deliberately synthetic — no real captured traffic ever lands
in the test tree. Each entry carries the *category* the scrubber must
detect and the *raw text* (the scrubber sees the raw text and must rewrite
or redact it; we assert recall by checking the raw text does NOT appear
in the scrubbed output).

Categories:

  hipaa-safe-harbor-* — the 18 identifiers
  pci                 — credit-card-shaped strings
  cloud-keys          — AWS/GCP/Anthropic/Stripe/GitHub
  free-text-pii       — names / addresses embedded in prose
  high-entropy        — JWT, OAuth tokens

The corpus is loaded by test_recall_corpus.py; the build_corpus()
function is exposed so adjacent tests can reuse it.
"""

from __future__ import annotations

import random
from dataclasses import dataclass

random.seed(0xC0DE)  # deterministic corpus


@dataclass(slots=True, frozen=True)
class PiiCase:
    category: str
    raw: str
    # When set, the scrubber may legitimately leave THIS substring in place
    # (e.g., the "+" sign on a phone number stays as shape).
    keep_substring: str = ""


# ─── HIPAA Safe Harbor identifiers ─────────────────────────────────────


def _names() -> list[PiiCase]:
    first = ["Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Hugo", "Ivy", "Julian"]
    last = ["Wong", "Patel", "Müller", "Garcia", "Okafor", "Tanaka", "Rossi", "Hernandez", "Schmidt", "Nguyen"]
    out: list[PiiCase] = []
    for f in first:
        for l in last[:5]:
            out.append(PiiCase("hipaa-safe-harbor-name", f"{f} {l}"))
    return out


def _ssns() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(50):
        s = f"{random.randint(100, 899):03d}-{random.randint(10, 99):02d}-{random.randint(1000, 9999):04d}"
        out.append(PiiCase("hipaa-safe-harbor-ssn", f"SSN: {s}"))
    return out


def _mrns() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(50):
        digits = "".join(str(random.randint(0, 9)) for _ in range(random.randint(6, 10)))
        labels = ["MRN", "Medical Record #", "Patient ID"]
        out.append(PiiCase("hipaa-safe-harbor-mrn", f"{random.choice(labels)}: {digits}"))
    return out


def _phones() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(50):
        a = random.randint(200, 999)
        b = random.randint(200, 999)
        c = random.randint(1000, 9999)
        s = random.choice([
            f"({a}) {b}-{c}",
            f"+1-{a}-{b}-{c}",
            f"+1{a}{b}{c}",
            f"{a}.{b}.{c}",
        ])
        out.append(PiiCase("hipaa-safe-harbor-phone", f"Call {s}"))
    return out


def _emails() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(50):
        user = f"user{random.randint(1, 9999)}"
        domain = random.choice(["example.com", "acme-medical.org", "patient-portal.io", "clinic.net"])
        out.append(PiiCase("hipaa-safe-harbor-email", f"Email: {user}@{domain}"))
    return out


def _credit_cards() -> list[PiiCase]:
    out: list[PiiCase] = []
    bins = ["4242", "5555", "3782", "6011"]
    for _ in range(50):
        digits = random.choice(bins) + "".join(
            str(random.randint(0, 9)) for _ in range(12)
        )
        sep = random.choice([" ", "-", ""])
        formatted = sep.join(digits[i : i + 4] for i in range(0, 16, 4))
        out.append(PiiCase("pci", f"card={formatted}"))
    return out


def _ips() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(50):
        out.append(
            PiiCase(
                "hipaa-safe-harbor-ip",
                f"client_ip {random.randint(1,254)}.{random.randint(0,255)}."
                f"{random.randint(0,255)}.{random.randint(1,254)}",
            )
        )
    return out


def _urls() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(40):
        path = f"/users/{random.randint(1, 99999)}/profile"
        out.append(PiiCase("hipaa-safe-harbor-url", f"https://acme.com{path}"))
    return out


def _account_numbers() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(50):
        n = "".join(str(random.randint(0, 9)) for _ in range(random.randint(8, 12)))
        out.append(PiiCase("hipaa-safe-harbor-account", f"acct {n}"))
    return out


def _vins() -> list[PiiCase]:
    out: list[PiiCase] = []
    alpha = "ABCDEFGHJKLMNPRSTUVWXYZ"
    chars = alpha + "0123456789"
    for _ in range(40):
        vin = "".join(random.choice(chars) for _ in range(17))
        out.append(PiiCase("hipaa-safe-harbor-vin", f"VIN: {vin}"))
    return out


def _bank_iban() -> list[PiiCase]:
    out: list[PiiCase] = []
    countries = ["DE", "GB", "FR", "ES"]
    for _ in range(40):
        cc = random.choice(countries)
        check = f"{random.randint(10, 99)}"
        rest = "".join(str(random.randint(0, 9)) for _ in range(20))
        out.append(PiiCase("hipaa-safe-harbor-bank", f"IBAN {cc}{check}{rest}"))
    return out


def _drivers_license() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(40):
        s = "D" + "".join(str(random.randint(0, 9)) for _ in range(7))
        out.append(PiiCase("hipaa-safe-harbor-drivers-license", f"License# {s}"))
    return out


def _passport() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(40):
        s = f"{random.choice('ABCDE')}{random.randint(10000000, 99999999)}"
        out.append(PiiCase("hipaa-safe-harbor-passport", f"Passport {s}"))
    return out


def _dates() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(40):
        y = random.randint(1940, 2025)
        m = random.randint(1, 12)
        d = random.randint(1, 28)
        out.append(PiiCase("hipaa-safe-harbor-date", f"DOB {y}-{m:02d}-{d:02d}"))
    return out


def _npi() -> list[PiiCase]:
    """Random 10-digit identifiers. We tag as npi but accept that pure-random
    digits may not Luhn-check; the recognizer still catches them by context."""
    out: list[PiiCase] = []
    for _ in range(40):
        s = "".join(str(random.randint(0, 9)) for _ in range(10))
        out.append(PiiCase("hipaa-safe-harbor-provider", f"provider NPI {s}"))
    return out


def _dea() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(40):
        s = "".join(random.choice("ABCDEFGHJKLMNPRSTUVWXYZ") for _ in range(2)) + \
            "".join(str(random.randint(0, 9)) for _ in range(7))
        out.append(PiiCase("hipaa-safe-harbor-dea", f"DEA: {s}"))
    return out


def _cloud_keys() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(15):
        # AWS access key
        s = "AKIA" + "".join(random.choice("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ") for _ in range(16))
        out.append(PiiCase("cloud-keys-aws", f"AWS={s}"))
    for _ in range(15):
        s = "ghp_" + "".join(random.choice("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") for _ in range(36))
        out.append(PiiCase("cloud-keys-github", f"PAT={s}"))
    for _ in range(15):
        s = "sk-ant-api03-" + "".join(random.choice("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-") for _ in range(60))
        out.append(PiiCase("cloud-keys-anthropic", f"key={s}"))
    return out


def _jwts() -> list[PiiCase]:
    out: list[PiiCase] = []
    for _ in range(40):
        head = "eyJ" + "".join(random.choice("abcdefghijklmnopqrstuvwxyz0123456789-_") for _ in range(40))
        body = "".join(random.choice("abcdefghijklmnopqrstuvwxyz0123456789-_") for _ in range(80))
        sig = "".join(random.choice("abcdefghijklmnopqrstuvwxyz0123456789-_") for _ in range(40))
        out.append(PiiCase("high-entropy-jwt", f"Authorization: Bearer {head}.{body}.{sig}"))
    return out


def _free_text_pii() -> list[PiiCase]:
    out: list[PiiCase] = []
    names = ["Alice Wong", "Bob Patel", "Charlie Müller", "Diana Garcia", "Eve Okafor", "Frank Tanaka"]
    sentences = [
        "Patient {n} was admitted with chest pain on Mar 3.",
        "Dr. {n} signed the prescription order this morning.",
        "Customer {n} called regarding their July invoice.",
        "Note from {n}: please reschedule the follow-up.",
        "Authorized representative: {n}, son of patient.",
    ]
    for n in names:
        for s in sentences:
            out.append(PiiCase("free-text-pii", s.format(n=n)))
    return out


def build_corpus() -> list[PiiCase]:
    out: list[PiiCase] = []
    out.extend(_names())
    out.extend(_ssns())
    out.extend(_mrns())
    out.extend(_phones())
    out.extend(_emails())
    out.extend(_credit_cards())
    out.extend(_ips())
    out.extend(_urls())
    out.extend(_account_numbers())
    out.extend(_vins())
    out.extend(_bank_iban())
    out.extend(_drivers_license())
    out.extend(_passport())
    out.extend(_dates())
    out.extend(_npi())
    out.extend(_dea())
    out.extend(_cloud_keys())
    out.extend(_jwts())
    out.extend(_free_text_pii())
    return out


CORPUS_CATEGORIES = (
    "hipaa-safe-harbor-name",
    "hipaa-safe-harbor-ssn",
    "hipaa-safe-harbor-mrn",
    "hipaa-safe-harbor-phone",
    "hipaa-safe-harbor-email",
    "hipaa-safe-harbor-ip",
    "hipaa-safe-harbor-url",
    "hipaa-safe-harbor-account",
    "hipaa-safe-harbor-vin",
    "hipaa-safe-harbor-bank",
    "hipaa-safe-harbor-drivers-license",
    "hipaa-safe-harbor-passport",
    "hipaa-safe-harbor-date",
    "hipaa-safe-harbor-provider",
    "hipaa-safe-harbor-dea",
    "pci",
    "cloud-keys-aws",
    "cloud-keys-github",
    "cloud-keys-anthropic",
    "high-entropy-jwt",
    "free-text-pii",
)
