% Relaxed twin of routing_policy.dl: only the threshold values move.
Decl policy_anchor(Marker).
policy_anchor("ok").

Decl threshold(Name, Value).
threshold("auto", "0.70") :- policy_anchor("ok").
threshold("margin", "0.20") :- policy_anchor("ok").
threshold("review", "0.60") :- policy_anchor("ok").

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
