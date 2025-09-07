# spinitron-proxy

Developers using the Spinitron API must adhere to the following terms of service:

> Two rules that we hope you will follow in your use of the Spinitron API can impact design of your app.
>
> First, as a client of Spinitron you have an API key that allows access to the API. You may not delegate your access to third parties except for development purposes. In particular, you may not delegate your access to clients of your web or mobile apps. For example, a web or mobile client of yours may fetch data from your servers but they may not use your API key to access the Spinitron API. In other words, don’t build Spinitron API clients into your client scripts or mobile apps.
>
> Second, you should use a cache in the implementation of your web servers or mobile app back-end servers. For example, say you display part of the program schedule on a certain web page. It’s not ok if for every request of that page from visitors to your site, your server makes a request to the Spinitron API. The program schedule doesn’t change rapidly so you can assume that data you fetched a minute ago is still valid. So you should cache data you fetch from Spinitron for a while in case you need the same info again soon. Cacheing is good for your website visitors (faster page loads), reduces load on your and Spinitron’s servers, reduces Internet traffic (and therefore even reduces energy waste a little). How you implement the cache is up to you. Good cache implementations take into account the specific design of the app and how users are expected to behave.

With that in mind, this little server...

- forwards requests from a client (e.g. a mobile app) to Spinitron with the developer's API key protected
- is read-only i.e. it only accepts GET requests*
- includes an in-memory cache mechanism optimized for <https://github.com/dctalbot/spinitron-mobile-app>
- *exposes a POST endpoint (`/trigger/spins`) for use by Spinitron to let the app know when a new spin arrives in real-time. This can be secured with a password.
- hosts a SSE stream (`/spin-events`) to let downstream consumers know when new spins are posted, in real-time
  - works with [watchdog services](https://github.com/wbor-fm/wbor-api-watchdog) to forward new spins to a RabbitMQ exchange

## Cache strategy

### Individual resources

- When selecting an endpoint with an ID value e.g. `/spins/1`
- Query parameters are ignored
- TTL of 3 minutes

### Collections

- When selecting an endpoint that returns a list e.g. `/spins?`, `/spins?page=1`
- Query parameters are not ignored
- TTL depends on the collection:
  - `personas`: 5m
  - `shows`: 5m
  - `playlists`: 3m
  - `spins`: 30s
- Upon expiration, all caches for the same collection are invalidated e.g. When `/spins?page=1` expires, `/spins?page=3` is also invalidated (and vice-versa).

## How to deploy

The following architectures are supported: `linux/amd64`, `linux/arm/v7`, `linux/arm64`, `linux/ppc64le`, and `linux/s390x`.

Container-based services are supported by most cloud providers. The memory and CPU requirements are extremely minimal, so just pick the cheapest option.

1. Change the environment variables in the `Makefile` as desired:
   - `IMAGE_NAME`
   - `CONTAINER_NAME`
   - `NETWORK_NAME` (default: `spinitron-proxy-network`).
   - `APP_PORT` (defaults to exposing `4001` on the host, speaking to `8080` in the container)
   - `DOCKER_TOOL` (default: `docker`, but also works for `podman`)
2. Set the Spinitron API key and base URL variables in a `.env` file:

    Copy the `.env.sample` file to `.env` and set the following variables:

    ```bash
    SPINITRON_API_KEY=your_spinitron_api_key
    INSTALLATION_BASE_URL=http://your_spinitron_installation_base_url
    TRIGGER_PASSWORD=a_secure_password # optional, for /trigger/spins endpoint
    ```

    To generate a password, you can use a command like:

    ```bash
    python -c "import secrets; print(secrets.token_urlsafe(24))"
    ```

    Copy the generated password into the `TRIGGER_PASSWORD` variable in your `.env` file.

3. Run: `make` (after setting changing the environment variables in the `Makefile` as desired)
4. The app will be available at `http://your_spinitron_installation_base_url:4001` (or whatever port you set in the `Makefile`).

  Check that it's working by making some requests:

  ```bash
  curl "http://your_spinitron_installation_base_url:4001/api/spins"
  ```

  You should see a JSON response with the latest spins.

### Spinitron Metadata Push

To have Spinitron notify the proxy when a new spin is logged, you can use the `/trigger/spins` endpoint. In your Spinitron admin settings, under "Metadata Push", configure a channel with the following URL:

`POST https://your_proxy_url/trigger/spins`

To secure this endpoint, set the `TRIGGER_PASSWORD` in your `.env` file. Then, in Spinitron, use the `%pw%` token in the URL like this:

`POST https://your_proxy_url/trigger/spins?pw=%pw%`

Set the password that matches the `TRIGGER_PASSWORD` you set in the "Password" field of the Spinitron Metadata Push channel settings.

## Related Projects

- <https://github.com/dctalbot/react-spinitron>
- <https://github.com/dctalbot/spinitron-mobile-app>

## How to Develop

- Go (version specified in `go.mod`)
- A Spinitron API key

1. Make changes to the app
2. Run `SPINITRON_API_KEY=XXX INSTALLATION_BASE_URL=localhost go run .`
3. Make [some requests](https://spinitron.github.io/v2api/) e.g. `curl "localhost:8080/api/spins"`

## How to Test

Run:

```bash
go test -v ./...
```

## Known Issues/Quirks

- **SSE Event Specificity:** Server-Sent Events (SSE) for new spins are specifically tied to updates of the canonical `/api/spins` cache entry (i.e., the spins endpoint without additional query parameters). This reduces the number of duplicate SSE notifications. Consequently, updates to more specific spin queries (e.g., `/api/spins?count=10&fields=artist`) do *not* directly trigger their own SSEs. Consumers of the SSE stream should expect notifications primarily when the main `/api/spins` data is refreshed.
- **Client-Side Idempotency:** While the proxy tries to minimize redundant SSEs, it's a good practice for clients consuming these events to be idempotent – that is, designed to handle multiple signals for the same underlying data update without adverse effects (e.g., by checking the latest spin ID and de-duping before processing).
- **SSE Connection Scalability:** The proxy maintains an active connection and a Go channel for each connected SSE client. For deployments with an extremely large number of concurrent SSE listeners, resource usage (memory, connection handling) should be monitored. Alternative or supplementary solutions like a dedicated message broker might be considered for very high-scale scenarios.
- **Cache TTL for `/api/spins`:** The default TTL for the `/api/spins` cache is 30 seconds. If no new spin is posted and no trigger event occurs, clients will not receive an SSE until this cache naturally expires and is subsequently repopulated by a client request to `/api/spins`. The `/trigger/spins` endpoint can be used for more immediate cache refreshes and SSE broadcasts.
