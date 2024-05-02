# Lomo-backup - Cost saving 2-stages cloud backup solution

[![GitHub Actions](https://github.com/lomorage/lomo-backup/actions/workflows/main.yml/badge.svg)](https://github.com/lomorage/lomo-backup/actions?query=workflow%3AGo)


# Motivation
Photos/videos are very import personal assets and we want to store in our home instead of clouds. We developped lomorage application to self host our own google photo alternative solutions, which has met our main goal.

At the same time, backup is extremely important as we don't want to lost any photos by accident. Current lomorage application can daily backup to another disk, or NAS via rsync, but it is still hosted at home. We want diaster recovery ability. We want to implement a peer backup solution which allows me to back up photos / videos to my parents' or sisters' or brothers' or friends' home, which will be our final goal, but we need a fast reliable way before that solution really works. Seems storing into cloud like Glacier would be one alternative. The solution should meet below requirements:
1. price is as cheap as possible
2. run backup daily
3. run consistency check monthly and send me alert if cloud version is different from my local version
4. support multi sites backup

As I have local duplicate backup as well, and most time I access photos and videos from local service, so I seldom visit the cloud backup version.

# Cost Analysis
Let us calculate the cost using AWS Glacier. As of 2024/4/4

- $0.0036 per GB / Month
- $0.03 (PUT, COPY, POST, LIST requests (per 1,000 requests))
- $0.0004 (GET, SELECT, and all other requests (per 1,000 requests)) (need recalculate)

assuming I have 50,000 photos (2M each) + 5,000 videos (30M each)， total storage is 250G, and total price will be 250 * 0.0036 = $0.9 / month, and total upload costs will be 55000 * 0.03 = $16.5. (need recalculate: consistency check will mainly use GET and LIST API, price will be 55000 * 0.0004 = $22 / month)

assuming I have 250 new photos (2M each) + 50 videos (30M each) per month, total new storage is 2G, total price will be 2 * 0.0036 = $0.0072, and all PUT API cost will be 300 * 0.03 = $9. (need recalculate: consistency check will be 300 * 0.0004 = $0.12).

# 2 Stages Approach
There are too many small files, thus API operation becomes main cost comparing with real storage cost. So how about I pack all images and videos into one big ISO, just like I burnt one CD rom to backup content at old days. 

But ISO approach has one limitation is to append into new files, you need have original ISO file, so we need either keep one copy locally, or download when backup is needed. Either one requires extra cost. 

Since many cloud provider offers free storage tier option, we can use them as middle man or staging station before getting ready to make ISO and back up to Glacier. So called 2 stages approach can meet this need. - use free storage to store metadata and short term backup- use GDA to store permanent files.

For example, Google drive offers 15G free space, AWS offers 15G free storage, MS one drive offers 5G free space. If we use 10G to make one ISO, new cost for storage will be same, but all upload costs will be 250/10 * 0.03 = $0.75. Consistency check price will be 250/10 * 0.0004 = $0.01

Now we'll do one upload every 5 months. Only 1 API operation is needed, and cost can be ignore.

Workflow:
1. Daily back up to free storage firstly.
2. When reaching configured disk threshold, archive the files and make into ISO file, save into Glacier, delete backup ones from free storage
3. one metadata file or sqlite db file specifies which files are in which ISO file, or free storage

Pre-requsition commands:
- mkisofs: generate iso file

Features:

- :heavy_check_mark: pack all photos/videos into multiple ISOs and upload to Glacier
- :heavy_check_mark: metadata to track which file is in which iso
- [ ] backup files not in ISO to staging station, Google drive
- [ ] encrypt iso files before upload to Glacier, Google drive
- [ ] metadata to track which files are in staging station
- [ ] daemon running mode to watch folder change only, avoid scanning all folder daily
- [ ] daily consistency check on staging station
- [ ] monthly consistency check on Glacier. send email alert if anything is wrong.
- [ ] multi platform support: remove mkisofs requirements

# Support us
If you find Lomo-Backup is useful, please support us below:

<a href="https://www.buymeacoffee.com/lomorage" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

<a href="https://opencollective.com/lomoware/donate" target="_blank">
  <img src="https://opencollective.com/webpack/donate/button@2x.png?color=blue" width=300 />
</a>

Also welcome to try our free Photo backup applications. https://lomorage.com.

# Feature highlights
- Multipart upload to S3
- Resume upload if one part was fail
- Self define iso size

# Pre-requisition
## AWS Glacier API Access ID and Access Secret
AWS has good tutorial to get key. After you get it, you can set them into environment variable:

- AWS_DEFAULT_REGION
- AWS_SECRET_ACCESS_KEY
- AWS_ACCESS_KEY_ID

Note `AWS_DEFAULT_REGION` is to specify which region your upload will reside. You can also specify it when you run the upload command.

## Google Cloud API OAuth credentials and token
You can skip if you have tokens already. Below are my steps to get credentials. Note that most steps are from https://developers.google.com/drive/api/quickstart/go, but seems the guide missed some steps, thus I have to add these steps in case readers need them.

### Enable the API
Before using Google APIs, you need to turn them on in a Google Cloud project. You can turn on one or more APIs in a single Google Cloud project. In the Google Cloud console, enable the Google Drive API. https://console.cloud.google.com/flows/enableapi?apiid=drive.googleapis.com

### Configure the OAuth consent screen
If you're using a new Google Cloud project to complete this quickstart, configure the OAuth consent screen and add yourself as a test user. If you've already completed this step for your Cloud project, skip to the next section. Note that `add yourself as a test user` is very import to ensure oauth authorization success.

1. In the Google Cloud console, go to Menu menu > APIs & Services > OAuth consent screen. https://console.cloud.google.com/apis/credentials/consent
2. For User type select `External`, then click Create. Note that original guide specifies `Internal` User Type, but seems it won't allow me to select `Internal`, so I select `External`, and it works.
3. Complete the app registration form, then click Save and Continue.
4. For now, you can skip adding scopes and click Save and Continue.
5. Review your app registration summary. To make changes, click Edit. If the app registration looks OK, click Back to Dashboard.

### Authorize credentials for a desktop application
To authenticate end users and access user data in your app, you need to create one or more OAuth 2.0 Client IDs. A client ID is used to identify a single app to Google's OAuth servers. If your app runs on multiple platforms, you must create a separate client ID for each platform.

1. In the Google Cloud console, go to Menu menu > APIs & Services > Credentials. https://console.cloud.google.com/apis/credentials
2. Click Create Credentials > OAuth client ID.
3. Click Application type > Desktop app.
4. In the Name field, type a name for the credential. This name is only shown in the Google Cloud console.
5. Click Create. The OAuth client created screen appears, showing your new Client ID and Client secret.
6. Click OK. The newly created credential appears under OAuth 2.0 Client IDs.
7. Save the downloaded JSON file as `gdrive-credentials.json`, and move the file to your working directory.

Note that the redirect_uris in `gdrive-credentials.json` are `http://localhost`. I can not find any places to change it, so our app will use it as default.

### Get access token
`lomo-backup`` app has one command to get token. Its usage is as below 
```
$  ./lomob util gcloud-auth --help
NAME:
   lomob util gcloud-auth - 

USAGE:
   lomob util gcloud-auth [command options] [arguments...]

OPTIONS:
   --cred value           Google cloud oauth credential json file (default: "gdrive-credentials.json")
   --token value          Token file to access google cloud (default: "gdrive-token.json")
   --redirect-path value  Redirect path defined in credentials.json (default: "/")
   --redirect-port value  Redirect port defined in credentials.json (default: 80)
```

Notes:
 - If your redirect_url is `http://localhost:8080/auth/callback`, you can run `./lomob util gcloud-auth --redirect-port 8080 --redirect-path /auth/callback`
 - If your redirect port is `80`, you probably need run the tool with sudo permission if you see permission deny failure message `WARN[0000] Failed to listen 80: listen tcp :80: bind: permission denied`.
 - If you run `lomob` as sudo, `gdrive-token.json` may be saved as `root` user, and you can change to your own user if you like.

Below is output after you run `lomob`
```
$ sudo ./lomob util gcloud-auth 
Starting listen on http://localhost:80
Go to the following link in your browser then follow the instruction: 
https://accounts.google.com/o/oauth2/auth?access_type=offline&client_id=xxxx&redirect_uri=http%3A%2F%2Flocalhost&response_type=code&scope=https%3A%2F%2Fwww.googleapis.com%2Fauth%2Fdrive&state=state-token
```
Copy the link into your browser, and follow the google authorization steps, token will be retrieved if you see below messages
```
Handle google callback: /?state=state-token&code=xxxxxx&scope=https://www.googleapis.com/auth/drive
Start exchange: xxxx
Exchange success, saving token into gdrive-token.json
```

Once you get token, you can verify if it works or not via `./lomob list gdrive`

# Usage
## Overall options and sub commands

```
$ ./lomob --help
NAME:
   lomob - Backup files to remote storage with 2 stage approach

USAGE:
   lomob [global options] command [command options] [arguments...]

AUTHOR:
    <support@lomorage.com>

COMMANDS:
   scan     Scan all files under given directory
   iso      ISO related commands
   list     List scanned files related commands
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --db value                   Filename of DB (default: "lomob.db")
   --log-level value, -l value  Log level for processing. 0: Panic, 1: Fatal, 2: Error, 3: Warn, 4: Info, 5: Debug, 6: TraceLevel (default: 4)
   --help, -h                   show help
```
## Scan related
### Scan given directory

```
$ ./lomob scan --help
NAME:
   lomob scan - Scan all files under given directory

USAGE:
   lomob scan [command options] [directory to scan]

OPTIONS:
   --ignore-files value, --if value  List of ignored files, seperated by comman (default: ".DS_Store,._.DS_Store,Thumbs.db")
   --ignore-dirs value, --in value   List of ignored directories, seperated by comman (default: ".idea,.git,.github")
   --threads value, -t value         Number of scan threads in parallel (default: 20)
```
### List scanned directory

```
$ ./lomob list dirs -h
NAME:
   lomob list dirs - List all scanned directories
```

### List big files scanned currently
```
$ ./lomob list bigfiles -h
NAME:
   lomob list bigfiles - List big files

USAGE:
   lomob list bigfiles [command options] [arguments...]

OPTIONS:
   --file-size value, -s value  Minimum file size in the list result. KB=1000 Byte (default: "50MB")
```

## ISO related
### Create ISO

```
$ ./lomob iso create -h
NAME:
   lomob iso create - Group scanned files and make iso

USAGE:
   lomob iso create [command options] [iso filename. if empty, filename will be <oldest file name>--<latest filename>.iso]

OPTIONS:
   --iso-size value, -s value  Size of each ISO file. KB=1000 Byte (default: "5GB")
```

### List created ISOs
```
$ ./lomob iso list -h
NAME:
   lomob iso list - List all created iso files

$ ./lomob iso list
[0xc000415ea0 0xc000415f10]
ID    Name                          Size       Status                   Region    Bucket    Files Count    Create Time            Hash
1     2024-04-13--2024-04-20.iso    21.9 MB    Created, not uploaded                        7              2024-04-20 20:53:31    d5cd6b88e766d417995f715ddc03dd19450f74ecee9b2d6804d1e7c55559fb81
2     2024-04-11--2024-04-20.iso    14.0 MB    Created, not uploaded                        290            2024-04-20 20:53:32    b5474aeaacd7cd5fea0f41ba4ed18b298224031ff5ff008b9ff5a25fdcaea2b2
```

### Upload ISO