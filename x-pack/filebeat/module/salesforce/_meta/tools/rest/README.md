# REST API

This PoC script collects EventLogFile [Login Events] data from the Salesforce connected App using the REST API.

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
    ["EVENT_TYPE","TIMESTAMP","REQUEST_ID","ORGANIZATION_ID","USER_ID","RUN_TIME","CPU_TIME","URI","SESSION_KEY","LOGIN_KEY","USER_TYPE","REQUEST_STATUS","DB_TOTAL_TIME","BROWSER_TYPE","API_TYPE","API_VERSION","USER_NAME","TLS_PROTOCOL","CIPHER_SUITE","AUTHENTICATION_METHOD_REFERENCE","TIMESTAMP_DERIVED","USER_ID_DERIVED","CLIENT_IP","URI_ID_DERIVED","LOGIN_STATUS","SOURCE_IP" "Login","20211102121044.225","4fDtUhdb0OA75Vl1cJIA1-","00D5j000000VI3n","0055j000000utlP","75","28","/services/oauth2/token","","UsK4oAHW1UESFBIQ","Standard","","38461157","Jakarta Commons-HttpClient/3.1","","9998.0","","TLSv1.2","ECDHE-RSA-AES256-GCM-SHA384","","2021-11-02T12:10:44.225Z","0055j000000utlPAAQ","Salesforce.com IP","","LOGIN_NO_ERROR","35.168.189.83"]
    ```
    
## How it works

Query used in following script:

`SELECT EventType FROM EventLogFile WHERE EventType = 'Login' OR EventType = 'Logout'`


Token URL used for Authentication:

`https://login.salesforce.com/services/oauth2/token`


AuthStyleInParams sends the "client_id" and "client_secret" in the POST body as application/x-www-form-urlencoded parameters.
