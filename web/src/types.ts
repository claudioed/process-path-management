/** Mirrors process-path-management's own dto.go processPathResponse
 *  shape exactly -- see that file's doc comment ("domain structs never
 *  cross this boundary"). This is the frontend's copy of that same
 *  boundary contract. */
export interface ProcessPath {
  pathId: string;
  matchPrefix: string;
  direct: boolean;
  requiredCapabilities: string[];
  status: "ACTIVE" | "DEACTIVATED";
  createdAt: string;
  updatedAt: string;
}
