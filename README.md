# SmartThings - Ring Alarm v2 ![](https://github.com/asishrs/smartthings-ringalarmv2/workflows/Build/badge.svg) ![](https://github.com/asishrs/smartthings-ringalarmv2/workflows/Release/badge.svg) 

:loudspeaker: ​This is the `version 3 `of this application. If you are looking for older versions see the links below. No support for older versions. In case of issues, I recommend to update to the latest versions.
* [version 1](https://github.com/asishrs/smartthings-ringalarm)
* [version 2](v2.md) 

**Note:** This approach is tested using [SmartThings Classic App](https://support.smartthings.com/hc/en-us/articles/205380554-Things-in-the-SmartThings-Classic-app). If you are on in new SmartThings app, let me know if this approach requires any changes. PRs are welcome!

------

<u>**SmartThings - Ring Alarm v2**</u>

- [Bridge Application](#bridge-application)
  - [Get the lambda binary](#get-the-lambda-binary)
    - [Download the latest version from Github Release (Recommended)](#download-the-latest-version-from-Github-Release-(Recommended))
    - [Build Go Binary from code](#Build-Go-Binary-from-code)
  - [Store the lambda to Amazon S3 Bucket](#Store-the-lambda-to-Amazon-S3-Bucket)
  - [Create AWS Resources and Deploy the Lambda](#Create-AWS-Resources-and-Deploy-the-Lambda.)
  - [Get your API Url and Key](#Get-your-API-Url-and-Key)
    - [Get Invoke URL](#Get-Invoke-URL)
    - [Get API Key](#get-api-key)
- [Test Lambda (Optional)](#test-lambda-(optional))
  - [URLs](#urls)
  - [Sample `meta` cURL](#sample-`meta`-curl)
  - [Sample `status` cURL](#sample-`status`-curl)
- [Setup Device Handler and Smart App](#setup-device-handler-and-smart-app)
- [Integration with webCoRE](#integration-with-webcore)
- [Licence](#license)

------



> - :clock1: This setup is going to take 30 minutes to an hour depending on your exposure on the [SmartThings app](https://docs.smartthings.com/en/latest/getting-started/first-smartapp.html), [AWS Lambda](https://aws.amazon.com/lambda/), and Go.
> - :dollar: Deploying the Bridge Application in AWS as a Lambda is free but you will be charged for the use of API Gateway and Data Transfer. 

This page explains, how to set up Ring Alarm as a virtual device on your SmartThings. Ring Alarm uses WebSockets to communicate to ring server for checking Alarm Status and Status changes. Unfortunately, SmartThings app does not support WebSockets, and we have to create a bridge application which accepts HTTP calls from SmartThings and communicate to Ring Alarm via WebSockets. Below diagram explains the flow.

![SmartThings - Ring Alarm](images/SmartThings-Ring.png?raw=true "SmartThings - Ring Alarm")

If you are still reading this, that means you are ready to invest at least an hour!!!

This setup requires the deployment of two different components.

## Bridge Application

As I mentioned before, the bridge application is a proxy between the SmartThings custom app and Ring Alarm. Bridge application will be deployed as an [AWS Lambda function](https://aws.amazon.com/lambda/) using Go. The `AWS Lambda function` will be exposed to the SmartThings App via a [Amazon API Gateway](https://aws.amazon.com/api-gateway/). To secure the api endpoint, this setup uses an [api-key](https://docs.aws.amazon.com/apigateway/api-reference/resource/api-key/). This setup uses [AWS Cloud​Formation](https://aws.amazon.com/cloudformation/) template to automatically create the required AWS resources and deploy the `lambda build` from [Amazon S3 Bucket](https://docs.aws.amazon.com/s3/index.html)

You need to have an active AWS account and the latest Lambda build from [here](https://github.com/asishrs/smartthings-ringalarmv2/releases) before proceeding to the next step. 

If you don't have an account, start [here](https://aws.amazon.com/account/) 

Follow steps below to setup the SmartThings Ring Alarm Lambda.

### Get the lambda binary

#### Download the latest version from Github Release (Recommended)

You can download the latest `deployment.zip` from [Github Release page](https://github.com/asishrs/smartthings-ringalarmv2/releases)


#### Build Go Binary from code

If you want to build the Lambda from the source code, you can do that by cloning this repo and then executing below steps. 

You have to install golang version 1.13 or higher for this.

````shell
> GOOS=linux go build -o main
> zip deployment.zip main
````

### Store the lambda to Amazon S3 Bucket

You need to store the `deployment.zip` file in an `amazon s3 bucket` so that the `cloud formation template` can use that for deployment.

Follow below steps to create a bucket and upload the `deployment.zip` file to that bucket.
1. Login to AWS Account and the navigate to https://s3.console.aws.amazon.com/s3/home?region=us-east-1 (You may be different region based on your account setup)
1. Click on **Create Bucket**
1. On **Name and region** page page enter name for your bucket as **st-ring-alarm** (You can change the name if you want, you will have an option to provide your bucket name during the stack setup later.)
1. Leave everything else as default values on **Name and region** page and click **next** button
1. Leave everything as default values on **Configure options** page and click **next** button
1. Leave everything as default values on **Set permissions** page and click **next** button
1. On **Review** page click on **Create Bucket**
1. Select the newly created bucket on https://s3.console.aws.amazon.com/s3/home?region=us-east-1 and click on **Upload** button
1. Upload the `deployment.zip` file either via **Drag and Drop** or by clicking on **Add Files** button. Leave all options as default on the upload page.

### Create AWS Resources and Deploy the Lambda.

You will be using [AWS Cloud​Formation template](aws/ringalarm-gateway.yaml) to create the stack. 

You need to have either this repository cloned or save a copy of [ringalarm-gateway.yaml](https://raw.githubusercontent.com/asishrs/smartthings-ringalarmv2/master/aws/ringalarm-gateway.yaml) file on your local before proceeding. 

1. Login to AWS Account and the navigate to https://console.aws.amazon.com/cloudformation/home?region=us-east-1 (You may be different region based on your account setup)
1. Click on **Create Stack** and choose **With new resources(standard)**
1. On the **Specify template** page choose **Upload a template file**
1. Click on **Choose file** and select the `ringalarm-gateway.yaml` from cloned repository or from download.
1. Click **Next**
1. On the **Specify stack details** page enter values
    1. Enter stack name as `st-ring-alarm` - You can use custom names if you want. 
    1. `apiStageName` - Leave as default
    1. `lambdaFunctionName` - Leave as default
    1. `s3BucketName` - If you have selected any names other than `st-ring-alarm` for your `amazon s3 bucket` you need to update that here otherwise leave as default.
1. On **Specify stack details** page click **Next**
1. Leave everything as default on **Configure stack options** page and click **Next**
1. On **Review** page, scroll down to bottom and select *I acknowledge that AWS CloudFormation might create IAM resources.* and click on **Create Stack**
1. Wait for 2-5 minutes for the creation of the stack. You will see a status **CREATE_COMPLETE** once your stack is successfully created. 

### Get your API Url and Key

In  this step, you will get your API Invoke URL and API Key for SmartThings Application configuration.

#### Get Invoke URL

1. Login to AWS Account and the navigate to https://console.aws.amazon.com/apigateway/main/apis?region=us-east-1 (You may be different region based on your account setup) 
Under the API, select your API
1. Click on **st-ring-alarm-api** (If you have entered a custom stack name the name of the api will be `<your stack name>-api`)
1. Click on **Dashboard** under **API:st-ring-alarm-api** (If you have entered a custom stack name the name of the api will be `API:<your stack name>-api`)
1. You can see **Invoke URL** on top of the page, save it for SmartThings Application configuration

#### Get API Key

1. From the API main page, select **API Keys**
1. Select the key **st-ring-alarm-apikey** (If you have entered a custom stack name the name of the key will be `<your stack name>-apikey`)
1. Click on **Show** link on the API key
1. Save API Key for SmartThings Application configuration.

### Test Lambda (Optional)

You can test your lambda before proceeding to next steps
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

## Setup Device Handler and Smart App
Follow the steps [here](https://github.com/asishrs/smartthings)

## Integration with webCoRE

You can add **Ring Alarm** to the *Which alarms and sirens* in the webCoRE and use like below in the pistons.

```
execute
	if
		Ring Alarm's status is 'home'
	then
		with
			Your Device
        do
        	Turn Off;
        end with;
    end if;
end execute;
```

## License

SmartThings - Ring Alarmv2 is released under the [MIT License](https://opensource.org/licenses/MIT).
