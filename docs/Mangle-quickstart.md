# Mangle Datalog — Deep Quick Start

*(Facts, Rules, Graphs, Records, Built-ins, Debugging)*

## 0) What you write in Mangle

* **Declarations**: `Decl pred(… ).` — tell Mangle what predicates (and arities) exist.
* **Facts (EDB)**: `pred(arg1, arg2, ...).` — base data.
* **Rules (IDB)**: `head :- body1, body2, ... .` — derived data; Mangle iterates to a fixpoint.

Everything ends with a period `.`

---

## 1) Predicates & Arity (why they matter)

A **predicate** is identified by **name + arity** (number of arguments).

* If you declare `edge(U, V)` (i.e., `edge/2`), you **must** always use 2 arguments everywhere: in facts, in rule heads, and in rule bodies.
* Using `edge/3` or `edge/1` elsewhere will cause arity errors.

**Good**

```prolog
Decl edge(U, V).
edge(/a, /b).
edge(/b, /c).
```

**Bad (mixed arity)**

```prolog
Decl edge(U, V).
edge(/a, /b, /c).  % ❌ Wrong: declared edge/2 but used edge/3
```

---

## 2) Constants & Types

Mangle has a few literal kinds you’ll use all the time:

* **Name constants** (a.k.a. atoms): start with `/`

  * Examples: `/a`, `/doc1`, `/color`
  * Great for identifiers and labels (think “symbols”, not strings).
* **Strings**: `"hello"`, `"doc1"`
* **Numbers**: `42`, `0.5`
* **Records**: `{ /key: value, /other: value2 }` (unordered key–value map)

**Tip:** prefer **name constants** for IDs (`/doc1`), and **strings** for free text (“Quarterly revenue report”).

---

## 3) Facts (EDB)

Facts are the raw data. Example graph:

```prolog
Decl node(N).
Decl edge(U, V).

node(/a).  node(/b).  node(/c).  node(/d).
edge(/a, /b).
edge(/b, /c).
edge(/a, /d).
```

---

## 4) Rules (IDB): how derivations work

Rules derive new facts from existing ones.

```prolog
Decl path(X, Y).

% direct reachability
path(X, Y) :- edge(X, Y).

% transitive closure (recursion)
path(X, Z) :- path(X, Y), edge(Y, Z).
```

* Variables start with **uppercase** (`X`, `Y`, `Z`).
* Commas are logical **AND**.
* Mangle runs rules repeatedly until no new facts appear (a **fixpoint**).
* Optional reflexivity:

  ```prolog
  path(X, X) :- node(X).
  ```

**Expected derived facts**

```
path(/a,/b), path(/b,/c), path(/a,/d), path(/a,/c)
```

---

## 5) Records & pattern matching (structured data)

Records let you attach structured metadata (order doesn’t matter; keys match by name).

```prolog
Decl instr(N, Info).
Decl is_load(N).

instr(/p4, { /var: "tmp1", /instr: "load", /rhs: "b2", /member: ".ptr" }).

% wildcard `_` ignores fields you don't care about
is_load(N) :-
  instr(N, { /instr: "load", /var: _, /rhs: _, /member: _ }).
```

**Notes**

* The pattern must match the **keys you provide**. Extra keys in the fact are okay.
* You can “pin” a field (`/instr: "load"`) and ignore the rest with `_`.

**Extracting a field value**
You don’t destructure values directly into variables from a record (there’s no “/var: V” variable capture to use outside that record). Typical patterns:

* Check for **presence/value** (`/instr: "load"`).
* Carry the entire record around as a value (e.g., `label(N, Info)`).

---

## 6) Negation & stratification (the safe way)

You can use **negation-as-failure** `:not P(...)`. Ensure your program is **stratified**: no cycles that depend on a negated predicate.

**Valid: negation on a base predicate**

```prolog
Decl employed(Person).
Decl unemployed(Person).

unemployed(P) :-
  person(P),
  :not employed(P).
```

**Avoid** rules where `A` depends (directly or indirectly) on `:not A(...)`. That’s non-stratified.

---

## 7) Built-ins you can rely on (and what to avoid)

**Common numeric built-ins** (arity = 2):

* `:lt(X, Y)`  — X < Y
* `:le(X, Y)`  — X ≤ Y
* `:gt(X, Y)`  — X > Y
* `:ge(X, Y)`  — X ≥ Y

**Equality/Inequality:**

* `X = Y`, `X != Y`

**Negation:**

* `:not P(…)`

**Avoid:**

* Regex or string built-ins (`:matches`, `:contains`, `:regex`, …) — **not supported**.
  Do string checks outside and emit facts (e.g., `phone_valid/1`).

---

## 8) Policy patterns (deny/allow, thresholds, validation)

### 8.1 Threshold decision (numeric)

```prolog
Decl best_score(S).
Decl threshold(T).
Decl deny(R).

deny("low_confidence") :-
  best_score(S),
  threshold(T),
  :lt(S, T).
```

Facts for a quick test:

```prolog
threshold(0.5).
best_score(0.42).
```

### 8.2 Validate formats (no regex in Mangle)

Do regex in your host program; mark validity with facts. Mangle consumes the tags:

```prolog
Decl phone(Value).
Decl phone_valid(Value).
Decl deny(Reason).

deny("bad_phone_format") :-
  phone(X),
  :not phone_valid(X).
```

Facts to test:

```prolog
phone("123-456-7890").  phone_valid("123-456-7890").
phone("abc").           % no phone_valid("abc") -> will deny
```

---

## 9) Constructing graphs from higher-level control flow

You can synthesize `edge/2` from more semantic facts:

```prolog
Decl succ(B, Next).
Decl br(B, Info).     % { /true: T, /false: F }
Decl edge(U, V).

edge(B, N) :- succ(B, N).

edge(B, T) :- br(B, { /true: T, /false: _ }).
edge(B, F) :- br(B, { /true: _, /false: F }).
```

Facts:

```prolog
succ(/b1, /b2).
br(/b2, { /true: /b3, /false: /b4 }).
```

Derived edges:

```
edge(/b1,/b2), edge(/b2,/b3), edge(/b2,/b4)
```

---

## 10) End-to-end examples

### 10.1 Graph + Reachability + Labels + Policy

**`program.dlog`**

```prolog
% ---- Declarations ----
Decl node(N).
Decl edge(U, V).
Decl path(X, Y).
Decl label(N, Info).
Decl best_score(S).
Decl threshold(T).
Decl deny(Reason).
Decl is_red(N).

% ---- Facts ----
node(/a). node(/b). node(/c).
edge(/a, /b).
edge(/b, /c).
label(/a, { /color: "red" }).

threshold(0.5).
best_score(0.42).

% ---- Rules ----
path(X, Y) :- edge(X, Y).
path(X, Z) :- path(X, Y), edge(Y, Z).

is_red(N) :- label(N, { /color: "red" }).

deny("low_confidence") :-
  best_score(S),
  threshold(T),
  :lt(S, T).
```

**What you’ll get**

* Reachability: `path(/a,/c)`
* Tag: `is_red(/a)`
* Policy: `deny("low_confidence")` because `0.42 < 0.5`

---

### 10.2 “Chat with data”-style: select one doc via a filter

*(No external framework—just the logic part.)*

**`select.dlog`**

```prolog
% Declarations
Decl user_attribute(Key, Value).   % e.g., ("doc_id","A123")
Decl doc_id(Doc, Id).              % ("doc1","A123")
Decl requested_doc_id(Id).
Decl target_doc(Doc).
Decl query_filter(Key, Value).

% Rules: route to the right doc
requested_doc_id(DocID) :- user_attribute("doc_id", DocID).
target_doc(D)           :- requested_doc_id(Req), doc_id(D, Req).

% Emit filter “instructions” for a downstream retriever
query_filter("id", D)       :- target_doc(D).
query_filter("doc_id", Req) :- requested_doc_id(Req).
```

**`kb.facts`**

```prolog
doc_id("doc1", "A123").
doc_id("doc2", "B456").
user_attribute("doc_id", "A123").
```

**Derived**

```
requested_doc_id("A123")
target_doc("doc1")
query_filter("id","doc1")
query_filter("doc_id","A123")
```

You can then consume `query_filter/2` outside (e.g., to pick the right file).

---

## 11) Debugging guide (error → cause → fix)

**“could not find predicate p(A0)”**

* You used `p/arity` in a rule body, but there’s **no `Decl p(...)`** (or no head/facts).
* **Fix**: Add `Decl p(...)` and at least one **fact** or a rule **head** that can derive it.

**“cannot redeclare … previous Decl …”**

* You declared the **same predicate/arity twice** (maybe in two files).
* **Fix**: Keep a single `Decl` for each predicate/arity across all loaded files.

**“number of arguments … does not match the mode []”**

* You called a **non-existent builtin** (e.g., `:matches(…)`) so Mangle thinks it has arity 0.
* **Fix**: Remove unsupported built-ins; use supported numeric comparisons; push regex to the host layer and emit facts.

**No results from a rule you expect**

* Check **arity mismatches**.
* Check **constants**: `/doc1` ≠ `"doc1"`.
* Ensure **all predicates in the body** have facts/derivations.
* If using negation, ensure the program is **stratified**.

---

## 12) Organization tips

* Put **all `Decl`** near the top of each file.
* Split by domain: `graph.dlog`, `policy.dlog`, `kb.facts`, `select.dlog`.
* Keep “expensive” checks (regex, string ops) **outside** Mangle; represent results as **tags/facts** that rules can combine.

---

## 13) Copy-paste mini-cookbook

**Transitive closure**

```prolog
Decl e(U,V).  Decl tc(X,Y).
tc(X,Y) :- e(X,Y).
tc(X,Z) :- tc(X,Y), e(Y,Z).
```

**Same block**

```prolog
Decl block_of(N,B).  Decl in_same_block(X,Y).
in_same_block(X,Y) :- block_of(X,B), block_of(Y,B), X != Y.
```

**Min confidence guard**

```prolog
Decl score(S).  Decl min(S).  Decl ok.
ok :- score(S), min(M), :ge(S, M).
```

**Record key equality**

```prolog
Decl tagged(N,Info). Decl red(N).
red(N) :- tagged(N, { /color: "red" }).
```

**Route to one doc**

```prolog
Decl user_attribute(K,V). Decl doc_id(Doc,Id).
Decl target(Doc).

target(D) :- user_attribute("doc_id", Id), doc_id(D, Id).
```

---

If you stick to: **(1) correct `Decl`, (2) consistent arity, (3) supported built-ins only**, Mangle programs stay small, clear, and composable—perfect for graphs, policies, and simple “query planning” logic.
