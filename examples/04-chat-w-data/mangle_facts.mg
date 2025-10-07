% ===============================
% FACTS: chat-with-data (example dataset & session)
% File: mangle_facts.pl
% ===============================

% ---- Session / user attributes (example runtime values) ----
% In production, these are injected per-session by your app or IdP.
user_attribute("user_id", "alice").
user_attribute("role", "analyst").
user_attribute("department", "sales").
% Entitlement patterns you might carry from your IAM provider:
user_attribute("doc_id", "A123").             % Direct entitlement to a specific doc
user_attribute("purpose", "analytics").       % Declared purpose for access (optional)

% ---- Documents present in the data lake / warehouse ----
doc(doc1).
doc_id(doc1, "A123").
doc_department(doc1, "sales").
doc_confidentiality(doc1, "normal").
doc_owner(doc1, "data-team").
doc_tag(doc1, "customers").

doc(doc2).
doc_id(doc2, "B456").
doc_department(doc2, "marketing").
doc_confidentiality(doc2, "high").
doc_owner(doc2, "marketing-team").
doc_tag(doc2, "leads").

doc(doc3).
doc_id(doc3, "S777").
doc_department(doc3, "sales").
doc_confidentiality(doc3, "restricted").
doc_owner(doc3, "sales-ops").
doc_tag(doc3, "pipeline").

% ---- Columns (schema) & sensitivity flags ----
% Use one fact per column; mark sensitive columns explicitly.
column(doc1, "customer_name").
column(doc1, "email").
column(doc1, "revenue").
column(doc1, "notes").
sensitive_column(doc1, "email").
sensitive_column(doc1, "notes").

column(doc2, "lead_name").
column(doc2, "email").
column(doc2, "score").
sensitive_column(doc2, "email").

column(doc3, "account").
column(doc3, "deal_size").
column(doc3, "owner").
column(doc3, "notes").
sensitive_column(doc3, "notes").

% ---- Optional: function-level or action-level allowlists ----
allowed_agg("sum").
allowed_agg("avg").
allowed_agg("count").

% ---- Optional: join relationships (for cross-doc chat) ----
% joinable(DocA, ColA, DocB, ColB).
joinable(doc1, "customer_name", doc3, "account").
