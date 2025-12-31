Feature: PII Protection Policy
  As a security officer
  I want to prevent PII data from being sent to external LLMs
  So that we maintain data privacy compliance

  Scenario: Block LLM calls with PII labels
    Given the user has "pii" label
    When calling "llm_generate"
    Then halt with "PII leakage detected"

  Scenario: Block LLM calls with sensitive labels
    Given the entity is labeled "sensitive"
    When calling "llm_generate"
    Then halt with "Sensitive data cannot be sent to external LLM"

  Scenario: Allow LLM calls with public data
    Given the entity is labeled "public"
    When calling "llm_generate"
    Then route to "llm_service"
