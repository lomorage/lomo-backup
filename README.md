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

Development phase 1:
1. pack all photos/videos into multiple ISOs and upload to Glancier
2. metadata to track which file is in which iso
   
Development phase 2:
1. daily backup new files to staging station
2. metadata to track which new files are in staging station

Development phase 3:
1. auto pack new files into ISO, and archive to Glancier
2. metadata to be updated for new location

Development phase 4:
1. daily consistency check on staging station
2. monthly consistency check on Glancier3. send email alert if anything is wrong.

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
