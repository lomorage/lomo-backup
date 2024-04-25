# Motivation
Photos/videos are very import personal assets and we want to store in our home instead of clouds. We developped lomorage application to self host our own google photo alternative solutions, which has met our main goal.

At the same time, backup is extremely important as we don't want to lost any photos by accident. Current lomorage application can daily backup to another disk, or NAS via rsync, but it is still hosted at home. We want diaster recovery ability. We want to implement a peer backup solution which allows me to back up photos / videos to my parents' or sisters' or brothers' or friends' home, which will be our final goal, but we need a fast reliable way before that solution really works. Seems storing into cloud like Glancier would be one alternative. The solution should meet below requirements:
1. price is as cheap as possible
2. run backup daily
3. run consistency check monthly and send me alert if cloud version is different from my local version
4. support multi sites backup

As I have local duplicate backup as well, and most time I access photos and videos from local service, so I seldom visit the cloud backup version.

# Cost Analysis
Let us calculate the cost using AWS Glancier. As of 2024/4/4

- $0.0036 per GB / Month
- $0.03 (PUT, COPY, POST, LIST requests (per 1,000 requests))
- $0.0004 (GET, SELECT, and all other requests (per 1,000 requests)) (need recalculate)

assuming I have 50,000 photos (2M each) + 5,000 videos (30M each)， total storage is 250G, and total price will be 250 * 0.0036 = $0.9 / month, and total upload costs will be 55000 * 0.03 = $16.5. (need recalculate: consistency check will mainly use GET and LIST API, price will be 55000 * 0.0004 = $22 / month)

assuming I have 250 new photos (2M each) + 50 videos (30M each) per month, total new storage is 2G, total price will be 2 * 0.0036 = $0.0072, and all PUT API cost will be 300 * 0.03 = $9. (need recalculate: consistency check will be 300 * 0.0004 = $0.12).

# 2 Stages Approach
There are too many small files, thus API operation becomes main cost comparing with real storage cost. So how about I pack all images and videos into one big ISO, just like I burnt one CD rom to backup content at old days. 

But ISO approach has one limitation is to append into new files, you need have original ISO file, so we need either keep one copy locally, or download when backup is needed. Either one requires extra cost. 

Since many cloud provider offers free storage tier option, we can use them as middle man or staging station before getting ready to make ISO and back up to Glancier. So called 2 stages approach can meet this need. - use free storage to store metadata and short term backup- use GDA to store permanent files.

For example, Google drive offers 15G free space, AWS offers 15G free storage, MS one drive offers 5G free space. If we use 10G to make one ISO, new cost for storage will be same, but all upload costs will be 250/10 * 0.03 = $0.75. Consistency check price will be 250/10 * 0.0004 = $0.01

Now we'll do one upload every 5 months. Only 1 API operation is needed, and cost can be ignore.

Workflow:
1. Daily back up to free storage firstly.
2. When reaching configured disk threshold, archive the files and make into ISO file, save into Glancier, delete backup ones from free storage
3. one metadata file or sqlite db file specifies which files are in which ISO file, or free storage

Pre-requsition commands:
- mkisofs: generate iso file
- mount / umount: validate ISO and print all files

Features:

- :heavy_check_mark: pack all photos/videos into multiple ISOs and upload to Glancier
- [ ] metadata to track which file is in which iso
- [ ] backup files not in ISO to staging station
- [ ] metadata to track which files are in staging station
- [ ] daemon running mode to watch folder change only, avoid scanning all folder daily
- [ ] daily consistency check on staging station
- [ ] monthly consistency check on Glancier3. send email alert if anything is wrong.
- [ ] multi platform support: port to windows and avoid mount/tree command to print tree structure

# Support us
If you find Lomo-Backup is useful, please support us below:

<a href="https://www.buymeacoffee.com/lomorage" target="_blank"><img src="https://cdn.buymeacoffee.com/buttons/v2/default-yellow.png" alt="Buy Me A Coffee" style="height: 60px !important;width: 217px !important;" ></a>

<a href="https://opencollective.com/lomoware/donate" target="_blank">
  <img src="https://opencollective.com/webpack/donate/button@2x.png?color=blue" width=300 />
</a>

Also welcome to try our free Photo backup applications: 

# Feature highlights
- Multipart upload to S3
- Resume upload if one part was fail
- Self define iso size

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