package httpapi

import "net/http"

// registerSegmentExportRoutes is the hook of the administrator export step (ticket
// #56): GET /api/admin/segment-corrections/export. It is empty until then; that step
// replaces this file's body only.
func (s *Server) registerSegmentExportRoutes(mux *http.ServeMux) {}
