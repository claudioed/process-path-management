Feature: Deactivating process paths
  As an operator of the process path management service
  I want to deactivate (retire) a process path
  So that downstream consumers stop accepting new work against it

  Background:
    Given the Process Path Management service is running

  @bdd
  Scenario: Deactivating an active path succeeds
    Given a process path "SLAM" is already defined with matchPrefix "slam" and capabilities "slam"
    When the process path "SLAM" is deactivated
    Then the deactivation is accepted with status 204
    And the process path "SLAM" now has status "DEACTIVATED"

  @bdd
  Scenario: Deactivating an already-deactivated path is idempotent
    Given a process path "PICK" is already defined with matchPrefix "pick" and capabilities "pick"
    And the process path "PICK" is already deactivated
    When the process path "PICK" is deactivated
    Then the deactivation is accepted with status 204

  @bdd
  Scenario: Deactivating a process path that was never defined returns 404
    When the process path "MISSING" is deactivated
    Then the deactivation is rejected with status 404
