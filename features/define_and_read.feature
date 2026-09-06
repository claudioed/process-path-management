Feature: Defining and reading process paths
  As an operator of the process path management service
  I want to define, list, and read process paths
  So that fulfillment-execution, wes-work-planning, and workforce-management
  eventually have a single, operator-configurable source of process-path
  definitions instead of a static YAML catalogue

  Background:
    Given the Process Path Management service is running

  @bdd
  Scenario: Defining a process path succeeds
    When a process path "PICK" is defined with matchPrefix "pick" and capabilities "pick"
    Then the request is accepted with status 201
    And the process path response reports pathId "PICK", matchPrefix "pick", and status "ACTIVE"

  @bdd
  Scenario: Getting a defined process path returns it
    Given a process path "PACK" is already defined with matchPrefix "pack" and capabilities "pack"
    When the process path "PACK" is requested
    Then the request is accepted with status 200
    And the process path response reports pathId "PACK", matchPrefix "pack", and status "ACTIVE"

  @bdd
  Scenario: Getting a process path that was never defined returns 404
    When the process path "MISSING" is requested
    Then the request is rejected with status 404

  @bdd
  Scenario: Defining a process path with a duplicate id is rejected
    Given a process path "SLAM" is already defined with matchPrefix "slam" and capabilities "slam"
    When a process path "SLAM" is defined with matchPrefix "slam" and capabilities "slam"
    Then the request is rejected with status 409

  @bdd
  Scenario: Listing process paths only returns active ones by default
    Given a process path "PICK" is already defined with matchPrefix "pick" and capabilities "pick"
    And a process path "PACK" is already defined with matchPrefix "pack" and capabilities "pack"
    And the process path "PACK" is already deactivated
    When the process paths are listed
    Then the request is accepted with status 200
    And the process path list reports 1 path
    When all process paths are listed including deactivated ones
    Then the process path list reports 2 paths
