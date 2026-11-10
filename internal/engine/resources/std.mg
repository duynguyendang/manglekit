% --- Manglekit Standard Library (v2.0) ---
% Auto-loaded on engine startup.

% ==========================================
% 1. DATA REFLECTION (DO NOT REMOVE)
% These predicates allow the engine to read JSON inputs and Graph data.
% ==========================================

% JSON Primitives (Flattened)
Decl json_str(Parent, Key, Val).
Decl json_num(Parent, Key, Val).
Decl json_bool(Parent, Key, Val).
Decl json_link(Parent, Key, Child).
Decl json_null(Parent, Key).

% Knowledge Graph Primitives (N-Quads/Triples)
Decl quad(S, P, O, G).
Decl triple(S, P, O).
triple(S, P, O) :- quad(S, P, O, _).

% ==========================================
% 2. SYSTEM CONTROL (Standard Vocabulary)
% ==========================================

% Arity 1: Global/Single-Context (JSON) 
Decl deny(Reason). 
Decl halt(Reason).
Decl route(NextStep). 
Decl retry(Feedback).

% Arity 2: Entity-Specific (Graph/Batch) 
% Allows pinning the decision to a specific node (Entity). 
Decl deny(Entity, Reason). 
Decl halt(Entity, Reason).
Decl route(Entity, NextStep). 
Decl retry(Entity, Feedback).

% Arity 3: Tier-Attributed (Governance)
% The third argument is the governance tier (T0, T1, T2, T3).
% The solver queries this form first to resolve tier-aware conflicts.
% Rules using halt/3 are resolved by tier priority: T0 > T1 > T2 > T3.
Decl deny(Entity, Reason, Tier).
Decl halt(Entity, Reason, Tier).
Decl route(Entity, NextStep, Tier).
Decl retry(Entity, Feedback, Tier).

% Config is always Key-Value (Arity 2) or Entity-Key-Value (Arity 3) 
Decl config(Key, Value). 
Decl config(Entity, Key, Value).

% Semantic Alias for backward compatibility
halt(Reason) :- deny(Reason).
halt(Entity, Reason) :- deny(Entity, Reason).
halt(Entity, Reason, Tier) :- deny(Entity, Reason, Tier).

% ==========================================
% 3. CONTEXT INJECTION
% These predicates are injected by the runtime environment.
% ==========================================

Decl attempt(N).            % Current retry count (0, 1, 2...).
Decl meta(Key, Value).      % Envelope metadata (e.g., user_id, session_id).
Decl label(Tag).            % Security taint labels (e.g., "pii", "unsafe").
Decl action_operation(Entity, Op). % Action being performed (e.g., action_operation("Req", "llm_generate")).

% Telemetry Predicate (Arity 2: Entity, RuleID)
Decl violation_rule(Entity, RuleID).

% ==========================================
% 4. AGGREGATION (Mangle v0.4.0+)
% Mangle supports aggregation using the |> syntax with fn: functions.
% Example usage in policies:
%
%   % Count action operations per entity
%   action_count(Entity, Count) :-
%     action_operation(Entity, _)
%     |> do fn:group_by(Entity),
%        let Count = fn:count().
%
%   % Sum numeric metadata values
%   total_cost(Entity, Total) :-
%     meta(Entity, Cost),
%     json_num(_, "cost", Cost)
%     |> do fn:group_by(Entity),
%        let Total = fn:sum(Cost).
%
% Available aggregation functions:
%   - fn:count()        : Count rows in group
%   - fn:sum(Column)   : Sum column values
%   - fn:group_by(Col) : Group by column
%   - fn:group_map()   : Create map during aggregation (v0.4.0+)
%
% Note: Aggregation requires Mangle v0.4.0+ and is evaluated at query time.
% ==========================================

% ==========================================
% 5. TEMPORAL REASONING (NOT YET AVAILABLE)
% Mangle v0.4.0 does not yet include temporal reasoning features.
% The following is documented for when temporal support is added:
%
%   % Facts with time intervals (when available)
%   active_user("alice")@[2024-01-01, 2024-06-15].
%
%   % Query: Was active in last 30 days
%   recent_activity(User) :-
%     <-[0d, 30d] active_user(User).
%
% Temporal operators:
%   <-[0d, 30d] : Was true sometime in past (diamond-minus)
%   -[0d, 30d]  : Was true continuously in past (box-minus)
%   <+[0d, 30d] : Will be true in future (diamond-plus)
%   +[0d, 30d]  : Will be true continuously in future (box-plus)
%
% Special values: _ (unbounded), now (current time)
%
% Note: Requires factstore.NewTemporalStore() which is not yet available.
% ==========================================

% --- Helper: String Matching ---

% input_contains(Req, Keyword)
% Checks if the input text payload contains the keyword (case-insensitive if supported).
% NOTE: fn:contains is not supported in Mangle v0.3.0, so this helper is currently disabled.
% input_contains(Req, Keyword) :-
%    value(Req, "text", Text),
%    fn:contains(Text, Keyword).
