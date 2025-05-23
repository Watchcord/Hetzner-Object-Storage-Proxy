# Hetzner-Object-Storage-Proxy
Simple proxy to redirect a custom domain to Hetzner's Object Storage


## Configuring
| Environment variable | Default   | Description                                                                                                |
| -------------------- | --------- | ---------------------------------------------------------------------------------------------------------- |
| `PORT`               | `3000`    | Controls the port the proxy listens on                                                                     |
| `HETZNER_REGION`     | `nbg1`    | Controls which Hetzner object storage location to proxy requests to. Options: fsn1, nbg1, hel1             |
| `CACHE_AGE`          | `2629800` | Number of seconds. Used to control the `max-age` value in `Cache-Control` header (if added to the request) |


## Default behaviour
- Requests to `/<bucket>/<key>` (with key required) have the header `Cache-Control` appended with the value `public, max-age=CACHE_AGE`. This enables caching of the response for `CACHE_AGE` seconds.
- Requests to `/<bucket>/<key>?X-Amz-Algorithm=...` that are also a presigned URL, have the header `Cache-Control` appended with the value `private, no-store, no-cache, must-revalidate`. This bypasses caching of the response for presigned URLs. The proxy deems a URL as a presigned URL if it has the following query param keys: `X-Amz-Algorithm`, `X-Amz-Credential` and `X-Amz-Signature`.
- Bucket names are validated against the RegEx `^[a-z0-9]([a-z0-9-]{1,61}[a-z0-9])?$`.
- The host header is set as the Bucket URL when proxying to Hetzner.


## URLs
> [!NOTE]  
> In the below table, the host `cdn.watchcord.ai` is pointed to the proxy

| Proxy URL                                             | Proxy destination                                                    | Note                                                                                         |
| ----------------------------------------------------- | -------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| `cdn.watchcord.ai`                                    | `<region>.your-objectstorage.com`                                    |                                                                                              |
| `cdn.watchcord.ai/<bucket>/[key]`                     | `<bucket>.<region>.your-objectstorage.com/[key]`                     | `[key]` is optional. Using no key will redirect directly to the bucket instead of an object. |
| `cdn.watchcord.ai/<bucket>/<key>?X-Amz-Algorithm=...` | `<region>.your-objectstorage.com/<bucket>/<key>?X-Amz-Algorithm=...` | The proxy automatically detects presigned URLs and corrects the target destination.          |
| `/_internal/health`                                   | `200 OK`                                                             | Can be used for HTTP(S) based healthchecks.                                                  |
