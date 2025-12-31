Feature: Role-Based Access Control
  Enforce access control based on user roles and permissions

  Scenario: Admin can access sensitive operations
    Given the user has "admin" label
    When calling "delete_user"
    Then route to "admin_service"

  Scenario: Admin can modify system settings
    Given the user has "admin" label
    When calling "update_config"
    Then route to "admin_service"

  Scenario: Regular users cannot delete
    Given the user has "user" label
    When calling "delete_user"
    Then halt with "Insufficient permissions"

  Scenario: Regular users cannot modify settings
    Given the user has "user" label
    When calling "update_config"
    Then halt with "Insufficient permissions"

  Scenario: Guests have read-only access
    Given the user has "guest" label
    When calling "write_data"
    Then halt with "Read-only access"
