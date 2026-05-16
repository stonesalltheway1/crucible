// indexes.cypher — FalkorDB indexes per tenant graph
//
// FalkorDB supports openCypher index DDL with CREATE INDEX FOR ...
// Indexes are graph-scoped: each tenant_<id> graph carries its own.

CREATE INDEX FOR (c:Convention) ON (c.id);
CREATE INDEX FOR (c:Convention) ON (c.status);
CREATE INDEX FOR (c:Convention) ON (c.category);
CREATE INDEX FOR (c:Convention) ON (c.scope_file_glob);
CREATE INDEX FOR (c:Convention) ON (c.layer);
CREATE INDEX FOR (c:Convention) ON (c.stack_tag);
CREATE INDEX FOR (c:Convention) ON (c.valid_from);
CREATE INDEX FOR (c:Convention) ON (c.valid_to);

CREATE INDEX FOR (s:SourceRef) ON (s.kind);
CREATE INDEX FOR (s:SourceRef) ON (s.pr);
CREATE INDEX FOR (s:SourceRef) ON (s.incident_id);

CREATE INDEX FOR (i:Incident) ON (i.id);
CREATE INDEX FOR (a:ADR) ON (a.path);
CREATE INDEX FOR (f:File) ON (f.path);

// Full-text index over rule_nl for the LLM-judge "is this rule similar
// to anything already in the graph" novelty score.
CREATE FULLTEXT INDEX FOR (c:Convention) ON EACH [c.rule_nl];

// Relationship-property indexes for bi-temporal queries:
//   MATCH (c)-[r:REINFORCED_BY]->(s) WHERE r.recorded_at > $cutoff ...
CREATE INDEX FOR ()-[r:REINFORCED_BY]-() ON (r.recorded_at);
CREATE INDEX FOR ()-[r:VIOLATED_BY]-()   ON (r.recorded_at);
CREATE INDEX FOR ()-[r:SUPERSEDED_BY]-() ON (r.valid_from);
