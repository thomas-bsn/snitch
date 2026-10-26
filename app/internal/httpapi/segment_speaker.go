package httpapi

import "net/http"

// registerSegmentSpeakerRoutes is the hook of the segment speaker step (ticket #54):
// PUT .../segments/{index}/speaker. It is empty until then; that step replaces this
// file's body only.
func (s *Server) registerSegmentSpeakerRoutes(mux *http.ServeMux) {}
