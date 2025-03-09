# spinitron-proxy

Developers using the Spinitron API must adhere to the following terms of service:

> Two rules that we hope you will follow in your use of the Spinitron API can impact design of your app.
>
> First, as a client of Spinitron you have an API key that allows access to the API. You may not delegate your access to third parties except for development purposes. In particular, you may not delegate your access to clients of your web or mobile apps. For example, a web or mobile client of yours may fetch data from your servers but they may not use your API key to access the Spinitron API. In other words, don’t build Spinitron API clients into your client scripts or mobile apps.
>
> Second, you should use a cache in the implementation of your web servers or mobile app back-end servers. For example, say you display part of the program schedule on a certain web page. It’s not ok if for every request of that page from visitors to your site, your server makes a request to the Spinitron API. The program schedule doesn’t change rapidly so you can assume that data you fetched a minute ago is still valid. So you should cache data you fetch from Spinitron for a while in case you need the same info again soon. Cacheing is good for your website visitors (faster page loads), reduces load on your and Spinitron’s servers, reduces Internet traffic (and therefore even reduces energy waste a little). How you implement the cache is up to you. Good cache implementations take into account the specific design of the app and how users are expected to behave.

With that in mind, this little server...

- forwards requests from a client (e.g. a mobile app) to Spinitron with the developer's API key
- is read-only i.e. it only services GET requests
- includes an in-memory cache mechanism optimized for <https://github.com/dctalbot/spinitron-mobile-app>
- exposes a POST endpoint that Spinitron can use to let the app know when a new spin arrives
- hosts a SSE stream to let downstream consumers know when new data is available, in real-time

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

1. Change environment variables in the `Makefile`:
   - `IMAGE_NAME`
   - `CONTAINER_NAME`
   - `NETWORK_NAME` (default: `spinitron-proxy-network`).
   - `APP_PORT` (defaults to exposing `4001` on the host, speaking to `8080` in the container)
   - `DOCKER_TOOL` (default: `docker`, but also works for `podman`)
2. Set the Spinitron API key variable: `export SPINITRON_API_KEY=YOUR_KEY_HERE`
3. Run: `make`

## Related Projects

- <https://github.com/dctalbot/react-spinitron>
- <https://github.com/dctalbot/spinitron-mobile-app>

## How to Develop

### Requirements

- Go (version specified in `go.mod`)
- A Spinitron API key

1. Make changes to the app
2. Run `SPINITRON_API_KEY=XXX go run .`
3. Make [some requests](https://spinitron.github.io/v2api/) e.g. `curl "localhost:8080/api/spins"`
