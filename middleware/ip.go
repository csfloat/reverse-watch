package middleware

import "net/http"

func IP(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ip := r.Header.Get("CF-Connecting-IP")
		if ip != "" {
			r.RemoteAddr = ip
		}
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}
