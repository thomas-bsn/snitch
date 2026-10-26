package httpapi

import "net/http"

// registerSegmentRetranscriptionRoutes is the hook of the retranscription step
// (ticket #55). It is empty until then; that step replaces this file's body only.
func (s *Server) registerSegmentRetranscriptionRoutes(mux *http.ServeMux) {}
