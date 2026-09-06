import { ProcessPathsScreen } from "./screens/ProcessPathsScreen";

/** Exposed as process_path_mfe/App via Module Federation. Routed under
 *  /process-path/* by the shell.
 *
 *  process-path-management models a single flat ProcessPath resource
 *  (no sub-hierarchy the way facility-layout's Site->Zone->Aisle chain
 *  is), so this remote is deliberately a single screen rather than a
 *  sub-nav of several -- see ProcessPathsScreen's own doc comment. */
export default function App() {
  return <ProcessPathsScreen />;
}
