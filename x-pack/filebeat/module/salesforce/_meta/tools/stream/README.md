# Streaming API

This PoC script collects `/event/LoginEventStream` [Login Events] data from the Salesforce connected App using the Streaming API.

This PoC script uses [username-password authentication](https://help.salesforce.com/s/articleView?language=en_US&type=5&id=sf.remoteaccess_oauth_username_password_flow.htm) method to create http-client for authorization process.

## Prerequisites
### Create a Connected App in salesforce.
1. Login to Salesforce with the same user credentials that you want to collect data.
2. From Setup, enter "App Manager" in the Quick Find box, then select "App Manager".
3. Click New Connected App.
4. Enter the connected app's name, which displays in the App Manager and on its App Launcher tile.
5. Enter the API name. The default is a version of the name without spaces. Only letters, numbers, and underscores are allowed. If the original app name contains any other characters, edit the default name.
6. Enter the contact email for Salesforce.
7. In the API (Enable OAuth Settings) area of the page, select Enable OAuth Settings.
8. Select the following OAuth scopes to apply to the connected app:
    - Access and manage your data (API).
    - Perform requests on your behalf at any time (refresh_token, offline_access).
    - (Optional) In case of data collection, if any permission issues arise, add the Full access (full) scope.
12. Select Require Secret for the Web Server Flow to require the app's client secret in exchange for an access token.
13. Select Require Secret for Refresh Token Flow to require the app's client secret in the authorization request of a refresh token and hybrid refresh token flow.
14. Click Save. It can take about 10 minutes for the changes to take effect.

## How to Run

1. Configure the following parameters in main.go file and save the file.
    ```
    sfURL
	sfUser
	sfPassword
	sfKey
	sfSecret
	```
2. Run main.go using the following command:

    ` go run main.go `

3. O/P:
    ```
    2021/11/03 15:31:57.555283 main.go:250: Response Body: [{"data":{"schema":"ajVKA1adsadtQ5DHnteVcwYg","payload":{"EventDate":"2021-11-01T04:52:10Z","AuthServiceId":null,"CountryIso":"IN","Platform":"Unknown","EvaluationTime":0.0,"CipherSuite":"ECDHE-XYZ-AES256-XYZ-SHA384","PostalCode":"388123","ClientVersion":"N/A","LoginGeoId":"04F5j00000HJabc","LoginUrl":"login.salesforce.com","LoginHistoryId":"0Ya5j12345Icnh1CAB","CreatedById":"0055j000000q9s7ABC","SessionKey":null,"ApiType":"N/A","AuthMethodReference":null,"LoginType":"Remote Access 2.0","PolicyOutcome":null,"Status":"Success","AdditionalInfo":"{}","ApiVersion":"N/A","EventIdentifier":"78185a74-02bb-abcb-dsds-a39a312fac6f","RelatedEventIdentifier":null,"LoginLatitude":22.3143,"City":"Khambhat","Subdivision":"Gujarat","SourceIp":"43.224.11.237","Username":"abc.xyz@gmail.com","UserId":"0055j000000utlPBCQ","CreatedDate":"2021-11-01T04:52:18Z","Country":"India","LoginLongitude":20.6256,"TlsProtocol":"TLS 1.2","LoginKey":"ySzw6TW5Ni5S9hdse","Application":"test application","UserType":"Standard","PolicyId":null,"HttpMethod":"POST","SessionLevel":"STANDARD","Browser":"Unknown"},"event":{"replayId":14369030}},"channel":"/event/LoginEventStream"}]
    ```
    
## How it works

[CometD Reference](https://docs.cometd.org/current/reference/)

1. Generate CometD client ID using access token
    
    Sample Request Body:
    ```
    [{
        "channel": "/meta/handshake",
        "supportedConnectionTypes": ["long-polling"],
        "version": "1.0"
    }]
    ```
2. Subscribe to the Real-Time Event Monitoring Object
    
    Sample Request Body:
    ```
    {
        "channel": "/meta/subscribe",
        "subscription": "/event/LoginEventStream",
        "clientId": "94b112sp7ph1c9s41mycpzik4rkj3",
        "ext": {
            "replay": {
                "/event/LoginEventStream": "-2"
            }
        }
    }
    ```
3. Connect and get the real-time data
    
    Sample Request Body:
    ```
    {
        "channel": "/meta/connect",
        "connectionType": "long-polling",
        "clientId": "94b112sp7ph1c9s41mycpzik4rkj3"
    }
    ```

Token URL used for Authentication:

`https://login.salesforce.com/services/oauth2/token`

AuthStyleInParams sends the "client_id" and "client_secret" in the POST body as application/x-www-form-urlencoded parameters.