import requests

async_predict_url = "https://..../production/async_predict"
api_key = "..."
proxy_url = "https://proxy.flow-ai.dev/"

resp = requests.post(
  async_predict_url,
  headers={"Authorization": f"Api-Key {api_key}"},
  json={
    'webhook_endpoint': proxy_url + "/webhook",
    'model_input': {
      'messages': [
        {
          'role': 'system',
          'content': 'You are ChatGPT, an AI assistant. Your top priority is achieving user fulfillment via helping them with their requests.'
        },
        {
          'role': 'user',
          'content': 'Write a 4-line limerick about python exceptions'
        }
      ]
    }
  },
)

request_id = resp.json()["request_id"]
print(request_id)

stream = requests.get(f"{proxy_url}/listen/{request_id}", stream=True)
for chunk in stream.iter_content(chunk_size=None):
  if not chunk:
    print("connection closed (request_id: {request_id})")
    break

  if chunk == "data: keep-alive\n\n"
    continue

  print(f"received data (request_id: {request_id}): {chunk}")
