package httpapi

import "net/http"

// registerSegmentHistoryRoutes is the hook of the history and restoration step
// (ticket #50): GET .../segments/{index}/history and POST .../segments/{index}/restore.
// It is empty until then; that step replaces this file's body only.
func (s *Server) registerSegmentHistoryRoutes(mux *http.ServeMux) {}
