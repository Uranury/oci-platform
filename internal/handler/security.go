package handler

import "github.com/gin-gonic/gin"

// contentSecurityPolicy only lets the page load and run what this binary
// serves. The built frontend has no inline scripts or <style> blocks (Vite
// emits files under /assets/, Svelte sets dynamic styles through the CSSOM,
// which CSP doesn't restrict), so neither 'unsafe-inline' nor 'unsafe-eval'
// is needed. The pen-stroke sound is a data URL, but it is decoded in JS,
// not fetched, so connect-src needs no data: either. If an asset Vite
// inlines (anything under its 4 KiB limit) ever shows up blocked, that is
// what to look at.
//
// `npm run dev` serves its own index.html from Vite, so this header only
// applies to the embedded build.
const contentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self'; " +
	"style-src 'self'; " +
	"img-src 'self'; " +
	"font-src 'self'; " +
	"connect-src 'self'; " +
	"media-src 'self'; " +
	"worker-src 'self'; " +
	"manifest-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'none'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

// securityHeaders sets the CSP, and nosniff so a response is only ever used
// as the type it was served as (an API JSON body can't be loaded as a
// script), on every response.
func securityHeaders(c *gin.Context) {
	h := c.Writer.Header()
	h.Set("Content-Security-Policy", contentSecurityPolicy)
	h.Set("X-Content-Type-Options", "nosniff")
	c.Next()
}
