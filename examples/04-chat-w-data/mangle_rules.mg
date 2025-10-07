% ===============================
% RULES: chat-with-data (policy logic)
% File: mangle_rules.pl
% ===============================

% ---------- Helpers ----------
privileged_user :-
    user_attribute("role", "admin");
    user_attribute("role", "security").

same_department(Doc) :-
    user_attribute("department", Dept),
    doc_department(Doc, Dept).

% ---------- Row / object access (pre-retrieve) ----------
% Direct entitlement by exact doc_id match (your original pattern)
pre_retrieve(Doc) :-
    user_attribute("doc_id", ID),
    doc_id(Doc, ID).

% Departmental access (multi-tenant by department)
pre_retrieve(Doc) :-
    same_department(Doc),
    doc_confidentiality(Doc, "normal").

% Escalation for privileged roles (admins/security can see everything)
pre_retrieve(Doc) :-
    privileged_user,
    doc(Doc).

% Optional: block highly confidential docs unless privileged
deny_retrieve(Doc) :-
    doc_confidentiality(Doc, "restricted"),
    \+ privileged_user.

% Effective allow = pre_retrieve and not explicitly denied
can_retrieve(Doc) :-
    pre_retrieve(Doc),
    \+ deny_retrieve(Doc).

% ---------- Column-level controls (post-retrieve masking) ----------
% Columns visible when either not sensitive, or user is privileged
visible_column(Doc, Col) :-
    column(Doc, Col),
    \+ sensitive_column(Doc, Col),
    can_retrieve(Doc).

visible_column(Doc, Col) :-
    column(Doc, Col),
    sensitive_column(Doc, Col),
    privileged_user,
    can_retrieve(Doc).

% Mask any sensitive column for non-privileged users
masked_column(Doc, Col, "redact") :-
    column(Doc, Col),
    sensitive_column(Doc, Col),
    \+ privileged_user,
    can_retrieve(Doc).

% Example of dynamic masking based on purpose (analytics ⇒ hash PII)
masked_column(Doc, Col, "hash") :-
    column(Doc, Col),
    sensitive_column(Doc, Col),
    user_attribute("purpose", "analytics"),
    \+ privileged_user,
    can_retrieve(Doc).

% ---------- Function-level (query-shaping) controls ----------
% Allow only certain aggregations for non-privileged users
allow_function(Func) :-
    privileged_user;
    allowed_agg(Func).

% ---------- Top-level permission for "chat with data" ----------
% A user can "chat with" a doc if they can retrieve it and at least one
% column is visible after masking logic (prevents empty/meaningless answers).
can_chat_with_data(Doc) :-
    can_retrieve(Doc),
    visible_column(Doc, _).

% ---------- Optional: cross-document constraints for joins ----------
% Allow joins only if user can chat with both sides and an approved join exists.
allow_join(DocA, ColA, DocB, ColB) :-
    can_chat_with_data(DocA),
    can_chat_with_data(DocB),
    joinable(DocA, ColA, DocB, ColB).

% ---------- Defaults ----------
% If nothing matches, deny by default (closed world posture).
deny_retrieve(Doc) :-
    doc(Doc),
    \+ can_retrieve(Doc).

% ---------- Examples (queries you can run in REPL) ----------
% ?- can_chat_with_data(doc1).
% ?- can_chat_with_data(doc2).
% ?- visible_column(doc1, Col).
% ?- masked_column(doc1, Col, Mask).
% ?- allow_function("sum").
% ?- allow_join(doc1, "customer_name", doc3, "account").
