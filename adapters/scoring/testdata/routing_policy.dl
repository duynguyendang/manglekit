% Routing policy for the scoring adapter's integration test.
%
% Thresholds are RULE-defined against an invariant anchor: ReloadPolicy swaps
% rules atomically but bare facts survive the swap as base facts, so a
% fact-defined bar would leave the old value queryable next to the new one.
% scoring.ThresholdsFromPolicy rejects that ambiguity; this shape avoids it.

Decl policy_anchor(Marker).
policy_anchor("ok").

Decl threshold(Name, Value).
threshold("auto", "0.80") :- policy_anchor("ok").
threshold("margin", "0.20") :- policy_anchor("ok").
threshold("review", "0.70") :- policy_anchor("ok").

Decl confidence(Value).
Decl margin_value(Value).
Decl entropy_value(Value).

halt("Req", "unobserved_distribution_escalate", "T1") :-
    action_operation("Req", "route_ticket"),
    meta("unobserved_distribution", "true").

halt("Req", "confidence_below_review_floor_escalate", "T1") :-
    action_operation("Req", "route_ticket"),
    meta("below_review", "true").

halt("Req", "low_confidence_send_to_review", "T1") :-
    action_operation("Req", "route_ticket"),
    meta("below_auto", "true"),
    !meta("below_review", "true"),
    confidence(P),
    threshold("auto", T).

halt("Req", "no_valid_choice_escalate", "T1") :-
    action_operation("Req", "route_ticket"),
    meta("no_valid_choice", "true").
