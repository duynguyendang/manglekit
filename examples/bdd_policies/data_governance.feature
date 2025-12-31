Feature: Data Governance
  Comprehensive data governance policies for compliance and security

  Scenario: Prevent cross-border data transfer
    Given the metadata "data_region" is "EU"
    When calling "export_data"
    Then halt with "Cross-border data transfer not allowed"

  Scenario: Require encryption for sensitive operations
    Given the user has "sensitive" label
    And the metadata "encryption" is "none"
    When calling "store_data"
    Then halt with "Encryption required for sensitive data"

  Scenario: Allow encrypted storage
    Given the user has "sensitive" label
    And the metadata "encryption" is "aes256"
    When calling "store_data"
    Then route to "secure_storage"

  Scenario: Audit high-risk operations
    Given the user has "admin" label
    When calling "delete_database"
    Then retry with "Please confirm this high-risk operation"
