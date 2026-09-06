Feature: Revising process paths
  As an operator of the process path management service
  I want to revise an active process path's matchPrefix and required capabilities
  So that path definitions can evolve without changing a path's identity

  Background:
    Given the Process Path Management service is running

  @bdd
  Scenario: Revising an active path's matchPrefix and capabilities succeeds
    Given a process path "PICK" is already defined with matchPrefix "pick" and capabilities "pick"
    When the process path "PICK" is revised with matchPrefix "pick-zone-a" and capabilities "pick,hazmat"
    Then the request is accepted with status 200
    And the process path response reports pathId "PICK", matchPrefix "pick-zone-a", and status "ACTIVE"

  @bdd
  Scenario: Revising a process path that was never defined returns 404
    When the process path "MISSING" is revised with matchPrefix "missing" and capabilities "missing"
    Then the request is rejected with status 404

  @bdd
  Scenario: A deactivated path rejects a revision
    Given a process path "PACK" is already defined with matchPrefix "pack" and capabilities "pack"
    And the process path "PACK" is already deactivated
    When the process path "PACK" is revised with matchPrefix "pack-zone-b" and capabilities "pack"
    Then the request is rejected with status 422
