import json

def main_handler(event, context):
    if event["path"] == '/':
        response_message = 'root'
    elif event["path"] == '/stop':
        response_message = 'stop'
    elif event["path"] == '/start':
        response_message = 'start'
    else:
        response_message = None
    return {
        "statusCode": 200,
        "body": json.dumps(response_message)
    }
