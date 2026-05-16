// constraints.cypher — uniqueness + non-null gates on the procedural graph
//
// FalkorDB constraint support (openCypher CREATE CONSTRAINT).

CREATE CONSTRAINT ON (c:Convention) ASSERT c.id IS UNIQUE;
CREATE CONSTRAINT ON (c:Convention) ASSERT EXISTS(c.rule_nl);
CREATE CONSTRAINT ON (c:Convention) ASSERT EXISTS(c.category);
CREATE CONSTRAINT ON (c:Convention) ASSERT EXISTS(c.status);
CREATE CONSTRAINT ON (c:Convention) ASSERT EXISTS(c.tenant_id);
CREATE CONSTRAINT ON (c:Convention) ASSERT EXISTS(c.layer);

CREATE CONSTRAINT ON (i:Incident) ASSERT i.id IS UNIQUE;
CREATE CONSTRAINT ON (a:ADR)      ASSERT a.path IS UNIQUE;
CREATE CONSTRAINT ON (f:File)     ASSERT f.path IS UNIQUE;
