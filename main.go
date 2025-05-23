package main

import (
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`)

func main() {
	region := os.Getenv("HETZNER_REGION")
	if region == "" {
		region = "nbg1"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	cache := os.Getenv("CACHE_AGE")
	if cache == "" {
		cache = "2629800"
	}

	log.SetOutput(os.Stdout)
	r := gin.Default()

	// Ignore favicon requests
	r.GET("/favicon.ico", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	// For healthcheck
	r.GET("/_internal/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	// Proxy to https://<region>.your-objectstorage.com/
	r.Any("/", func(c *gin.Context) {
		targetHost := region + ".your-objectstorage.com"
		targetURL := "https://" + targetHost + "/"

		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to create request")
			return
		}

		copyHeaders(req.Header, c.Request.Header)
		applyForwardingHeaders(req, c, targetHost)
		req.URL.RawQuery = c.Request.URL.RawQuery

		proxyAndRespond(c, req)
	})

	// Proxy to https://<bucket>.<region>.your-objectstorage.com/<key>
	r.Any("/:bucket/*key", func(c *gin.Context) {
		bucket := c.Param("bucket")
		key := strings.TrimPrefix(c.Param("key"), "/")

		if !bucketNameRegex.MatchString(bucket) {
			c.String(http.StatusBadRequest, "Invalid bucket name")
			return
		}

		var targetHost, targetURL string
		if isPresignedRequest(c) {
			targetHost = region + ".your-objectstorage.com"
			targetURL = "https://" + targetHost + "/" + bucket + "/" + key
			log.Printf("Presigned URL: %s", targetURL)
		} else {
			targetHost = bucket + "." + region + ".your-objectstorage.com"
			targetURL = "https://" + targetHost + "/" + key
			log.Printf("Public URL: %s", targetURL)
		}

		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to create request")
			return
		}

		copyHeaders(req.Header, c.Request.Header)
		applyForwardingHeaders(req, c, targetHost)
		req.URL.RawQuery = c.Request.URL.RawQuery

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("Proxy error: %v", err)
			c.String(http.StatusBadGateway, "Proxy error: "+err.Error())
			return
		}
		defer resp.Body.Close()

		copyHeaders(c.Writer.Header(), resp.Header)

		if isPresignedRequest(c) {
			c.Writer.Header().Set("Cache-Control", "private, no-store, no-cache, must-revalidate")
		} else {
			c.Writer.Header().Set("Cache-Control", "public, max-age="+cache)
		}

		c.Status(resp.StatusCode)
		io.Copy(c.Writer, resp.Body)
	})

	log.Printf("Proxy server listening on :%s\n", port)
	r.Run(":" + port)
}

func copyHeaders(dst, src http.Header) {
	for name, values := range src {
		for _, value := range values {
			dst.Add(name, value)
		}
	}
}

func applyForwardingHeaders(req *http.Request, c *gin.Context, targetHost string) {
	req.Host = targetHost
	req.Header.Set("Host", targetHost)
	req.Header.Set("X-Forwarded-Host", c.Request.Host)
	req.Header.Set("X-Real-IP", clientIP(c))
	req.Header.Set("X-Forwarded-For", c.ClientIP())
}

func proxyAndRespond(c *gin.Context, req *http.Request) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Proxy error: %v", err)
		c.String(http.StatusBadGateway, "Proxy error: "+err.Error())
		return
	}
	defer resp.Body.Close()

	copyHeaders(c.Writer.Header(), resp.Header)

	c.Status(resp.StatusCode)
	io.Copy(c.Writer, resp.Body)
}

func clientIP(c *gin.Context) string {
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.ClientIP()
	}
	return ip
}

func isPresignedRequest(c *gin.Context) bool {
	q := c.Request.URL.Query()
	return q.Has("X-Amz-Algorithm") && q.Has("X-Amz-Credential") && q.Has("X-Amz-Signature")
}
