# SmartThings - Ring Alarm v2

![Build Status](https://api.travis-ci.org/asishrs/smartthings-ringalarmv2.svg?branch=master "Build Status")

:loudspeaker: ​This is the `version 2 `of this application. If you are looking for the `version 1`, visit https://github.com/asishrs/smartthings-ringalarm

**:arrow_up: Upgrading from v1? Check [here](#upgrading-from-v1)** 

> - :clock1: This setup is going to take 30 minutes to an hour depending on your exposure on the [SmartThings app](https://docs.smartthings.com/en/latest/getting-started/first-smartapp.html), [AWS Lambda](https://aws.amazon.com/lambda/), and Go.
> - :dollar: Deploying the Bridge Application in AWS as a Lambda is free but you will be charged for the use of API Gateway and Data Transfer. 

This page explains, how to set up Ring Alarm as a virtual device on your SmartThings. Ring Alarm uses WebSockets to communicate to ring server for checking Alarm Status and Status changes. Unfortunately, SmartThings app does not support WebSockets, and we have to create a bridge application which accepts HTTP calls from SmartThings and communicate to Ring Alarm via WebSockets. Below diagram explains the flow.

![SmartThings - Ring Alarm](images/SmartThings-Ring.png?raw=true "SmartThings - Ring Alarm")

**Note:** I have SmartThings classic app, and this approach is tested using that. If you are on in new SmartThings app, let me know if this approach requires any changes. PRs are welcome!

If you are still reading this,  that means you are ready to invest at least an hour!!!

This setup requires the deployment of two different components.

## Bridge Application
As I mentioned before, the bridge application is a proxy between the SmartThings custom app and Ring Alarm. For ease of deployment, I created this as an [AWS Lambda function](https://aws.amazon.com/lambda/) using Go.

You need to install this Lambda in AWS and set up an API gateway to communicate to that. This approach is using the API with Lambda integration using API Gateway. This code also requires an API authentication token. If you are already familiar with setting Lambda with API token, you can skip to the SmartThings Device Handler and Smart App.

Follow the below steps to install and setup Lambda in AWS. You need to have AWS  account and the latest Lambda build from [here](https://github.com/asishrs/smartthings-ringalarmv2/releases) before proceeding to the next step. If you don't have an account, start [here](https://aws.amazon.com/account/)

If you want to build the Lambda on your side, you can do that by cloning this repo and then executing below steps. 

#### Build Go Binary

You have to install golang version 1.12 and [dep](https://github.com/golang/dep) for this.

````shell
> dep ensure
> GOOS=linux go build -o main
> zip deployment.zip main
````

### Deploy a lambda in AWS?
#### Setup `Go` Function

- Open https://console.aws.amazon.com/lambda/home?region=us-east-1#/functions
- Click on **Create Function** and provide below details
  * **Name** - a name for your lambda (Example: Ring-Alarm)
  * **Runtime** - Select *Go 1.x*
  * **Role** - Select *Create new role from a template(s)*
  * **Role Name** - a name for the role (Example: ring-alarm-user)
  * **Policy templates** - Leave Empty
- Click on **Create function**. This process takes a few seconds.
- Once your function is ready, you will be directed to function settings page.
- On the **Designer** section, click on **API gateway** on the left side navigation.
- Configure API Gateway
  * **API** - Select *Create a new API*
  * **Security** - Select *Open with API Key*
  * Click on **Add**
- Click on **Save** button on right side top.
- On the **Designer** section, click on your function name.
- In the Function Code section, make sure you have values for **Upload a .zip file** as **Code Entry** **type** and **Go 1.x** as **Runtime**.
- Click on the **Upload** button and select the **deployment.zip** from [releases](https://github.com/asishrs/smartthings-ringalarmv2/releases) (latest version) or the local built version in project root directory.
- Update **Handler** as ***main***
- Click on **Save** button on right side top.

#### Update API

* Open https://us-east-1.console.aws.amazon.com/apigateway/home?region=us-east-1#/apis
* Under APIs, click on your API.
* From the Actions, select **Create Resource**
  * Enable Configure as proxy resource
  * Resource Path - Update value as **{ring-action+}**
  * Click on **Create Resource**
  * Lambda Function - Enter name of your Lambda function
  * Click on **Save**
  * Click in **Ok**

#### Enable API Key

* Click in **ANY** from the Resource **/{ring-action+}**
* Click on **Method Request** on the right-hand side.
  * Change **API Key Required** to *true* and click on the small **apply** icon.

#### Deploy API

* Under the API, select your API
* Click in **Resources**
* From the Actions, select **Deploy API**
* Select Deployment stage as **default**
* Click **Deploy**
* Save **Invoke URL** for SmartThings Application configuration

#### Get API Key

* From the API main page, select API Keys
* Select your API Key
* Click on **Show** link on the API key
* Save API Key for SmartThings Application configuration.

### Test Lambda

#### URLs

- POST /{Invoke URL From Above}/meta
- POST /{Invoke URL From Above}/status
- POST /{Invoke URL From Above}/off
- POST /{Invoke URL From Above}/home
- POST /{Invoke URL From Above}/away

#### Sample `meta` cURL

**Request**

```
  curl -X POST \
    {Invoke URL From Above}/meta \
    -H 'x-api-key: aws_gateway_api_key' \
    -d '{
    "user": "ring username",
    "password" : "ring password"
  }'
```

**Response**

```
{
    "locationId": "your_ring_alarm_locaion_id",
    "zId": "your_ring_alarm_zid"
}
```



#### Sample `status` cURL

Request with `locattionId` and `zid`. 

Eventhough you can run without locattionId` and `zid`,  this is the recommended approach as it will save time in the lambda execution

```
  curl -X POST \
    {Invoke URL From Above}/status \
    -H 'x-api-key: aws_gateway_api_key' \
    -d '{
    "user": "ring username",
    "password" : "ring password",
    "locationId" : "ring location Id",
    "zid" : "ring zid",
    "historyLimit": 10
  }'
```

Request without `locattionId` and `zid`. 

```
  curl -X POST \
    {Invoke URL From Above}/status \
    -H 'x-api-key: aws_gateway_api_key' \
    -d '{
    "user": "ring username",
    "password" : "ring password"
    "historyLimit": 10
  }'
```

## Get Ring Location Id and ZID

Ring Alarm requires to pass location id and zid of your alarm as part the web sockets call. Though this can achieve via API calls, we don't want to do that as this increases the total number of calls to make before actual web sockets call. The recommended way to get the `locationId` and `zid` is via the `/meta` api call (check sample request above). If you can't do that, follow below steps to get those values from the network panel of your browser. 

#### Location Id
- Open your **chrome network panel** (*Option + Command + I in Mac*) and login to Ring Alarm.
- In the network panel, search for **locations**.
- Click on the location API call on the left side.
- From the right side
  * In the **Header panel**, confirm the URL is https://app.ring.com/rhq/v1/devices/v1/locations
  * In the **Preview panel**, you can see the value of **location_id**. Save **location_id** for lambda testing and SmartThings Application configuration.

#### ZID
- *Optional*, open your **chrome network panel** (*Option + Command + I in Mac*) and login to Ring Alarm.
- In the network panel, search for **socket.io**
- Click on the WebSocket call on the left side.
- From the right side
  * In the **Frames** panel, check a frame response with message like `"msg":"DeviceInfoDocGetList"` (**Tip**: *If you are using chrome browser, you can see a red color down arrow on the left side of message.*)
  * Copy that value (*Right Click on the mouse and select **Copy Message***) and paste in your favorite text editor. I prefer an editor like Visual Studio Code as I can format that big message using JSON format.
  * Search for **Ring Alarm** on the message.
  * On that block, you can find a JSON key **zid**. Save **zid** for lambda testing and SmartThings Application configuration.

## Setup Device Handler and Smart App
Follow the steps [here](https://github.com/asishrs/smartthings)

## ​​Upgrading from V1

#### :arrow_up: Update `Go` Function 

- Open https://console.aws.amazon.com/lambda/home?region=us-east-1#/functions and select the `java` based version 1 function.
- On the function page go to **Function code** section and update below 
  - **Runtime** to `Go 1.x`
  - **Handler** to `main`
  - **Code entry type** should be `Upload a .zip file` (no changes)
  - Click in **Upload** button and choose  **deployment.zip** from [releases](https://github.com/asishrs/smartthings-ringalarmv2/releases) (latest version) or the local built version in project root directory. 
  - Click on **Save** button on right side top.
- Test the API - Refer [Test lambda](#test-lambda)

#### :arrow_up: Update Device Handler

Install the latest code for device handler from https://github.com/asishrs/smartthings/blob/master/devicetypes/asishrs/ringalarm.src/ringalarm.groovy

✏️ Update the number of ring devices in the code, check for below part.

```
//Define number of devices here.
def motionSensorCount = 5
def contactSensorCount = 6
def rangeExtenderCount = 1
def keypadCount = 1
```

## License

SmartThings - Ring Alarmv2 is released under the [MIT License](https://opensource.org/licenses/MIT).