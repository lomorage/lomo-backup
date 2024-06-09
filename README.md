# Lomo-backup - Cost saving 2-stages cloud backup solution

[![GitHub Actions](https://github.com/lomorage/lomo-backup/actions/workflows/main.yml/badge.svg)](https://github.com/lomorage/lomo-backup/actions?query=workflow%3AGo)

Lomo-backup is a backup solution designed to protect your valuable photos and videos using a two-stage approach. This strategy combines the benefits of free cloud storage with long-term archival in AWS Glacier.

# Motivation
Photos and videos are very important personal assets, and we prefer to store them at home rather than in the cloud. We developed the Lomorage application to self-host our own Google Photos alternative, successfully achieving our primary goal.

However, backups are crucial because we don't want to risk losing any photos accidentally. The current Lomorage application can perform daily backups to another disk or NAS via rsync, but these backups are still hosted at home. We need disaster recovery capabilities, making cloud storage a natural choice. Unfortunately, cloud storage is not always reliable; there are occasional reports of lost user data. Therefore, a solution that can run consistent checks monthly and send alerts if any discrepancies are found would be very beneficial.

Our initial plan was to implement a peer backup solution, allowing us to back up photos and videos to the homes of parents, siblings, or friends. While this remains our ultimate goal, a single backup copy is insufficient, and cloud storage typically offers a higher SLA than peer storage. Thus, the ideal solution should meet the following requirements:

- Cost-effectiveness
- Automatic backup when new files are detected
- Monthly consistency checks with alerts if the cloud version differs from the local version
- Support for multi-site backups

Since I already have local duplicate backups and usually access photos and videos from the local service, I rarely need to access the cloud backup version.

# Cost Analysis using current backup solution
Let us calculate the cost using AWS Glacier. As of 2024/4/4

- $0.0036 per GB / Month
- $0.03 (PUT, COPY, POST, LIST requests (per 1,000 requests))
- $0.0004 (GET, SELECT, and all other requests (per 1,000 requests)) (need recalculate)

assuming I have 50,000 photos (2M each) + 5,000 videos (30M each)， total storage is 250G, and total price will be 250 * 0.0036 = $0.9 / month, and total upload costs will be 55000 * 0.03 = $16.5. (need recalculate: consistency check will mainly use GET and LIST API, price will be 55000 * 0.0004 = $22 / month)

assuming I have 250 new photos (2M each) + 50 videos (30M each) per month, total new storage is 2G, total price will be 2 * 0.0036 = $0.0072, and all PUT API cost will be 300 * 0.03 = $9. (need recalculate: consistency check will be 300 * 0.0004 = $0.12).

# 2 Stages Approach
Due to the large number of image and video files, API operations become the primary cost compared to actual storage costs. To mitigate this, we propose packing all images and videos into a single large ISO file, similar to burning a CD-ROM for backup in the old days. By creating a 10GB ISO, the storage cost remains the same, but upload costs are minimized to $0.75 (calculated as 250/10 * 0.03). The consistency check cost is $0.01 (250/10 * 0.0004).

However, the ISO approach has a limitation: to append new files, you need the original ISO file. This means either keeping a local copy or downloading it when a backup is needed, both of which incur additional costs.

At the same time, many cloud providers offer free storage tiers. For example, Google Drive offers 15GB of free space, AWS offers 15GB, and Microsoft OneDrive offers 5GB.

To best utilize these servers, we created a "2-stage approach" solution:

- Using free storage for metadata and short-term backups as an intermediary or staging area before creating the ISO and backing it up to Glacier.
- Using Glacier deep archive for permanent file storage.

We plan to upload once every five months, requiring only one API operation, which is negligible in cost.

One note is I like to keep all photos/videos in remote backup when they are packed in ISO even I delete the ones at local because
1. Photos/videos will not be packed into ISO until total size of unpacked ones reach configured iso size, thus user have time to delete the ones they don't want
2. Number of deleted ones should not be that big, thus cost should be very small if storing in Glacier

Workflow:

1. Perform daily backups to free storage initially.
2. When reaching the configured disk threshold, archive the files into an ISO file, save it to Glacier, and delete the backup files from free storage.
3. Maintain a metadata file or SQLite database specifying which files are in which ISO file or free storage.

Pre-requsition commands:
- mkisofs: generate iso file

Features:

- :heavy_check_mark: pack all photos/videos into multiple ISOs and upload to S3
- :heavy_check_mark: metadata to track which file is in which iso
- :heavy_check_mark: backup files not in ISO to staging station, Google drive
- :heavy_check_mark: pack all photos/videos into multiple ISOs and upload to Glancier
- :heavy_check_mark: encrypt iso files before upload to Glacier, Google drive
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
- Checksum validation during upload
- Self define iso size
- On the fly encryption all files as iso file size may be big, and we want to avoid intermittent file in order to save time and not require extra disks
- Original file hash and encrypted file hash are kept in cloud for future consistency check

# Security Model
The security model is from repository [filecrypt](https://github.com/kisom/filecrypt). For more details, refer to the book [Practical Cryptography With Go](https://leanpub.com/gocrypto/read).

This program assumes that an attacker does not currently have access
to either the machine the archive is generated on, or on the machine
it is unpacked on. It is intended for medium to long-term storage of
sensitive data at rest on removeable media that may be used to load data
onto a variety of platforms (Windows, OS X, Linux, OpenBSD), where the
threat of losing the storage medium is considerably higher than losing a
secured laptop that the archive is generated on.

Key derivation is done by pairing a password with a randomly-chosen
256-bit salt using the scrypt parameters N=2^20, r=8, p=1. This makes
it astronomically unlikely that the same key will be derived from the
same passphrase. The key is used as a NaCl secretbox key; the nonce for
encryption is randomly generated. It is thought that this will be highly
unlikely to cause nonce reuse issues.

The primary weaknesses might come from an attack on the passphrase or
via cryptanalysis of the ciphertext. The ciphertext is produced using
NaCl appended to a random salt, so it is unlikely this will produce any
meaningful information. One exception might be if this program is used
to encrypt a known set of files, and the attacker compares the length of
the archive to a list of known file sizes.

An attack on the passphrase will most likely come via a successful
dictionary attack. The large salt and high scrypt parameters will
deter attackers without the large resources required to brute force
this. Dictionary attacks will also be expensive for these same reasons.

### Key & Salt & Nonce
`Lomo-backup` encrypts all data and metadata, in which listing the original filename, during upload and decrypts them on-the-fly upon retrieval. Each file upload has a unique encryption key derived from a master key. The master key is either input via the command line or derived from the environment variable `LOMOB_MASTER_KEY`. The key derivation function (KDF) used is Argon2, the winner of the Password Hashing Competition. Each file's salt for the KDF is the first 16 byte of its SHA256 checksum.

### Notes
- ISO and iso metadata filename is not encrypted

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
`lomo-backup`` app has one command utility to get token. Its usage is as below 
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

# Basic Backup Steps
The basic workflow is simple: 1. scan, 2. pack ISO, 3. upload ISO or files. Anytime you can list remote directories in cloud in tree view, and download and restore them.

1. **Install the software:** 
    ```sh
    go install github.com/lomorage/lomo-backup/cmd/lomob@latest
    ```
2. **Create a directory to store generated database file and iso files:** 
    ```sh
    mkdir lomo-backup
    cd lomo-backup
    ```
3. **Scan the directory containing images and videos:** 
    ```sh
    lomob scan ~/Pictures
    ```
4. **Create ISO files:** 
    ```sh
    lomob iso create
    ```
    - Default ISO file size is 5GB. Use `-s` to change this value.
    - Refer to the detailed usage section for larger sizes.
5. **Set AWS credentials:** Set them in the environment variables as per the previous section.
6. **Upload ISO files to AWS:** 
    ```sh
    lomob upload iso
    ```
    - Default upload part size is 100MB. Use `-p` to change this.
    - Files are encrypted by default. Use `--no-encrypt` to upload raw files.
    - Default storage class is S3 `STANDARD`. Use `--storage-class` to change to `DEEP_ARCHIVE`.
    - Refer to the detailed usage section for other settings.
7. **Get Google Cloud OAuth credentials:** Obtain the JSON file and token file.
8. **Upload unpacked files to Google Cloud:** 
    ```sh
    lomob upload files
    ```

# More Detail Usage
## Overall options and sub commands
```
$ ./lomob --help
NAME:
   lomob - Backup files to remote storage with 2 stage approach

USAGE:
   lomob [global options] command [command options] [arguments...]

COMMANDS:
   scan     Scan all files under given directory
   iso      ISO related commands
   upload   Upload packed ISO files or individual files
   restore  Restore encrypted files cloud
   list     List scanned files related commands
   util     Various tools
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --db value                   Filename of DB (default: "lomob.db")
   --log-level value, -l value  Log level for processing. 0: Panic, 1: Fatal, 2: Error, 3: Warn, 4: Info, 5: Debug, 6: TraceLevel (default: 4)
   --help, -h                   show help
```

## Scan Folder
Specify one starting folder to scan. Files under the directories will be added into a sqlite db. For example, `lomob scan /home/scan/workspace/golang/src/lomorage/lomo-backup`. `--ignore-files` and `--ignore-dirs` will skip the specified files and directories.
```
$ lomob scan -h
NAME:
   lomob scan - Scan all files under given directory

USAGE:
   lomob scan [command options] [directory to scan]

OPTIONS:
   --ignore-files value, --if value  List of ignored files, separated by comma (default: ".DS_Store,._.DS_Store,Thumbs.db")
   --ignore-dirs value, --in value   List of ignored directories, separated by comma (default: ".idea,.git,.github")
   --threads value, -t value         Number of scan threads in parallel (default: 20)
```

## Create ISO
`lomob iso create` will automatically pack all files into ISOs. If total size of files are beyond iso size, it will recreate a new ISO file and continue packing process. Default ISO size is 5G, but you can specify your own.
```
$ lomob iso create -h
NAME:
   lomob iso create - Group scanned files and make iso

USAGE:
   lomob iso create [command options] [iso filename. if empty, filename will be <oldest file name>--<latest filename>.iso]

OPTIONS:
   --iso-size value, -s value   Size of each ISO file. KB=1000 Byte (default: "5G")
   --store-dir value, -p value  Directory to store the ISOs. It's urrent directory by default
```

## Upload

Note that the name of first folder under given bucket is the scan root directory whose name made by this formular:
- split the full path into different parts
- rejoin all parts with `_`
- for example, if scan root directory full path is `/home/scan/workspace/golang/src/lomorage/lomo-backup`, the folder name will be `home_scan_workspace_golang_src_lomorage_lomo-backup`

### 3.1 Upload ISOs to AWS
You can either specify the actual ISO files to upload, or if no filenames are provided, it will upload all the created ISO files.
```
$ lomob upload iso -h
NAME:
   lomob upload iso - Upload specified or all iso files

USAGE:
   lomob upload iso [command options] [arguments...]

OPTIONS:
   --awsAccessKeyID value         aws Access Key ID [$AWS_ACCESS_KEY_ID]
   --awsSecretAccessKey value     aws Secret Access Key [$AWS_SECRET_ACCESS_KEY]
   --awsBucketRegion value        aws Bucket Region [$AWS_DEFAULT_REGION]
   --awsBucketName value          awsBucketName (default: "lomorage")
   --part-size value, -p value    Size of each upload partition. KB=1000 Byte (default: "6M")
   --nthreads value, -n value     Number of parallel multi part upload (default: 3)
   --save-parts, -s               Save multiparts locally for debug
   --no-encrypt                   not do any encryption, and upload raw files
   --force                        force to upload from scratch and not reuse previous upload info
   --encrypt-key value, -k value  Master key to encrypt current upload file [$LOMOB_MASTER_KEY]
   --storage-class value          The  type  of storage to use for the object. Valid choices are: DEEP_ARCHIVE | GLACIER | GLACIER_IR | INTELLIGENT_TIERING | ONE-ZONE_IA | REDUCED_REDUNDANCY | STANDARD | STANDARD_IA. (default: "STANDARD")
```


### 3.2 Upload files not packaged in ISOs to google drive
```
$ lomob upload files -h
NAME:
   lomob upload files - Upload individual files not in ISO to google drive

USAGE:
   lomob upload files [command options] [arguments...]

OPTIONS:
   --cred value                   Google cloud oauth credential json file (default: "gdrive-credentials.json")
   --token value                  Token file to access google cloud (default: "gdrive-token.json")
   --folder value                 Folders to list (default: "lomorage")
   --encrypt-key value, -k value  Master key to encrypt current upload file [$LOMOB_MASTER_KEY]
```

## 4. List
### 4.1 List scanned directory
```
$ ./lomob list dirs -h
NAME:
   lomob list dirs - List all scanned directories
```

### 4.2 List big files scanned currently
```
$ ./lomob list bigfiles -h
NAME:
   lomob list bigfiles - List big files

USAGE:
   lomob list bigfiles [command options] [arguments...]

OPTIONS:
   --file-size value, -s value  Minimum file size in the list result. KB=1000 Byte (default: "50MB")
```

### 4.3 List created ISOs
This command list all created isos, and also display their current status, uploaded or not, upload success or failed, etc
```
$ lomob list iso -h
NAME:
   lomob iso list - List all created iso files

$ lomob list iso
[0xc000415ea0 0xc000415f10]
ID    Name                          Size       Status                   Region    Bucket    Files Count    Create Time            Hash
1     2024-04-13--2024-04-20.iso    21.9 MB    Uploaded                 us-east-1 lomorage  7              2024-04-20 20:53:31    d5cd6b88e766d417995f715ddc03dd19450f74ecee9b2d6804d1e7c55559fb81
2     2024-04-11--2024-04-20.iso    14.0 MB    Created, not uploaded                        290            2024-04-20 20:53:32    b5474aeaacd7cd5fea0f41ba4ed18b298224031ff5ff008b9ff5a25fdcaea2b2
```

### 4.3 List files in one ISOs
You can list all files in one ISO in tree view
```
$ lomob iso dump 2024-04-13--2024-04-28.iso
/
├── [   04/20/2024]  clients
│   └── [               6900    04/27/2024]  upload.go
├── [   04/28/2024]  cmd
│   └── [       04/28/2024]  lomob
│       ├── [           6340    04/28/2024]  iso.go
│       ├── [           4270    04/24/2024]  list.go
│       └── [       24584368    04/28/2024]  lomob
├── [            915    04/26/2024]  go.mod
├── [           4407    04/26/2024]  go.sum
├── [           7450    04/23/2024]  README.md
└── [            506    04/13/2024]  gitignore
```
### 4.4 List files not in any isos
This command is to list which files not packed into ISOs, and also show if it is uploaded in cloud or not
```
$ lomob list files
In Cloud    Path
Y           /home/scan/workspace/golang/src/lomorage/lomo-backup/common/testdata/indepedant_declaration.txt
Y           /home/scan/workspace/golang/src/lomorage/lomo-backup/vendor/github.com/aws/aws-sdk-go/private/protocol/eventstream/debug.go
Y           /home/scan/workspace/golang/src/lomorage/lomo-backup/vendor/golang.org/x/sys/windows/types_windows_arm.go
Y           /home/scan/workspace/golang/src/lomorage/lomo-backup/vendor/golang.org/x/sys/windows/types_windows_arm64.go```
```

### 4.5 List files in google drive
You can run below command to list directories in tree view in google drive. It has 4 fields in front of each file name: 
- file size in Byte
- file mod time get through os.Stat.ModTime
- first 6 letters of file original sha256 hash
- first 6 letters of encrypted file sha256 hash

```
$ lomob list gdrive
lomorage
└── [   05/03/2024]  home_scan_workspace_golang_src_lomorage_lomo-backup
    └── [       05/09/2024]  common
        └── [   04/26/2024]  testdata
            ├── [               8163    04/26/2024      4cfd75  be28f7]  indepedant_declaration.txt
```

You can also restore any files
### 5. Restore
### 5.1 Restore files in google drive
```
$ lomob restore gdrive -h
NAME:
   lomob restore gdrive - Restore files in google drive

USAGE:
   lomob restore gdrive [command options] [encrypted file name in fullpath] [output file name]

OPTIONS:
   --cred value                   Google cloud oauth credential json file (default: "gdrive-credentials.json")
   --token value                  Token file to access google cloud (default: "gdrive-token.json")
   --encrypt-key value, -k value  Master key to encrypt current upload file [$LOMOB_MASTER_KEY]
```
### 5.2 Restore isos in AWS S3
```
$ lomob restore aws -h
NAME:
   lomob restore aws - Restore ISO files in AWS drive

USAGE:
   lomob restore aws [command options] [iso file name] [output file name]

OPTIONS:
   --awsAccessKeyID value         aws Access Key ID [$AWS_ACCESS_KEY_ID]
   --awsSecretAccessKey value     aws Secret Access Key [$AWS_SECRET_ACCESS_KEY]
   --awsBucketRegion value        aws Bucket Region [$AWS_DEFAULT_REGION]
   --awsBucketName value          awsBucketName (default: "lomorage")
   --encrypt-key value, -k value  Master key to encrypt current upload file [$LOMOB_MASTER_KEY]
```

## Utility tools
### Acquire Google oauth credentail json file
```
$ lomob util gcloud-auth -h
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

### Refresh Google Oauth Token
```
$ lomob util gcloud-auth-refresh -h
NAME:
   lomob util gcloud-auth-refresh - 

USAGE:
   lomob util gcloud-auth-refresh [command options] [arguments...]

OPTIONS:
   --cred value   Google cloud oauth credential json file (default: "gdrive-credentials.json")
   --token value  Token file to access google cloud (default: "gdrive-token.json")
```

## License
This software is released under GPL-3.0.