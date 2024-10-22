## API documentation

## `POST /token`

**Generates token required for connecting to `/listen` stream for specific Baseten request ID.**

Tokens expire after 15 minutes.

Expected request body:

```json
{ "request_id": "«request id»" }
```

### Example request

```shell
curl -XPOST localhost:8000/token --data '{"request_id": "7cb1e320-cbcf"}'
```

### Success response

- **Response status code:** `200`
- **Response body:**
    ```json
    { "token": "[0-9a-f]{32}", "expires_at": "«unix timestamp»" }
    ```

### Error – missing or malformed body / missing or incorrect request ID

- **Response status code:** `400`
- **Response body:** ```Bad request. Field `request_id` (string) is required.```

### Error – token already generated and not expired for given request ID

- **Response status code:** `409`
- **Response body:** ```token already exists```

### Error – internal server error

- **Response status code:** `500`
- **Response body:** ```cannot generate token```

---

## `GET /listen/:request_id`

**Opens and maintains HTTP [SSE stream](https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events) for clients
to receive Baseten webhook responses.**

Connection requires an earlier generated token in the `Authorization` header.
Upon successful connection, the server will start sending a series of events.
The connection with the client will be automatically dropped after timeout specified in `-timeout` runtime
flag.
The server sends the following headers to start the SSE connection:

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

### Example request

```shell
curl localhost:8000/listen/7cb1e320-cbcf -H'Authorization: Bearer 123456789...abcdef'
```

### Successful connection – events sent by the server

* Keep-alive event, sent every 5 seconds to keep the connection open
  ```
  data: keep-alive\n\n
  ```
* Server gone event. Sent when proxy process is signaled to stop or when the connection times out
  ```
  data: server gone\n\n
  ```
* Webhook payload. Sent when webhook payload from Baseten is delivered.
  ```
  data: «json response»\n\n
  ```
* Webhook payload signature. The value of `X-BASETEN-SIGNATURE` header of the original Baseten webhook request
  ```
  data: signature=«signature»\n\n
  ```
* "End of transmission", sent after the payload and signature are sent
  ```
  data: eot\n\n
  ```

### Error – lack of `Authorization` header or invalid or expired token

- **Response status code:** `401`
- **Response body:** ```unauthorized```

### Error – internal server error when opening the SSE connection

- **Response status code:** `500`
- **Response body:** ```failed to open stream, try again later```

### Error – internal server error when retrieving webhook response

- **Response status code:** `500`
- **Response body:** ```failed to retrieve response```

---

## `POST /webhook`

**Endpoint to which the Baseten webhooks payloads are delivered.**

Expected request body: as described in
[Baseten documentation](https://docs.baseten.co/invoke/async#processing-async-predict-results).

### Example request

```shell
curl -XPOST localhost:8000/webhook \
  --data '{"request_id": "7cb1e320-cbcf", "model_id": "my_model_id", ...}' \
  -H 'X-BASETEN-SIGNATURE: ....'
```

### Success response

- **Response status code:** `200`
- **Response body:** (empty)

### Error – missing `X-BASETEN-SIGNATURE` header

- **Response status code:** `400`
- **Response body:** ```bad request```
-

### Error – invalid or malformed request body, missing required `request_id` field

- **Response status code:** `400`
- **Response body:** ```bad request```

### Error – internal server error

- **Response status code:** `500`
- **Response body:** ```internal server error```
