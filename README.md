# 🕵️‍♂️ Hunt

**Go Microservices For Bug Bounty**

Welcome to **Hunt**! This project is a collection of Go microservices designed to assist in bug bounty hunting by automating various security assessments. Once findings meet a certain criteria, I get notified by webhooks in my Discord server. 
The setup was designed to be **cost-effective**, maintaining continuous scanning for around **$100/month**.

**Potential Takeovers**
![image](https://github.com/user-attachments/assets/d6acb0a8-bd2b-4ed5-a4d5-ca2caac7bd10)


**Subdomain Enumeration**
![image](https://github.com/user-attachments/assets/4efe92b2-213f-4f2d-a95a-8ffb3e9e2955)


**Nuclei Scans**
![image](https://github.com/user-attachments/assets/c6f3b15e-c643-4931-b99a-4a1384c0ead3)


**LFI Scans**
![image](https://github.com/user-attachments/assets/0fdd8605-efc7-458b-9a35-26674ccb5c4c)


**Open Re-directs**
![image](https://github.com/user-attachments/assets/fe70ffaa-5a8f-4f31-ae9c-96c476352a9e)


**XSS Scans**
![image](https://github.com/user-attachments/assets/21f06fc4-a6ee-44d6-867b-fbc21d759407)



## 📂 Project Structure

The repository is organized into the following directories and scripts:

- **LFIDetection/**: Detects Local File Inclusion vulnerabilities.
- **MassDNS/**: Performs mass DNS resolution to discover subdomains. (automated scanner)
- **NucleiVulnerabilityScanning/**: Utilizes Nuclei for vulnerability scanning.
- **OpenRedirectCheck/**: Checks for open redirect vulnerabilities.
- **ParameterJSExtraction/**: Extracts parameters from JavaScript files.
- **PortScanning/**: Conducts port scanning to identify open ports.
- **SubdomainExploiting/**: Exploits discovered subdomains. (ad-hoc manual scanner - connects to database)
- **URLCollection/**: Collects URLs for further analysis.
- **XSSDetection/**: Detects Cross-Site Scripting vulnerabilities.

Additionally, the repository includes various `run_*.sh` scripts to execute the corresponding services via cronjobs (On your VPS's). The times can be changed in order to preserve IP's and also manually manage the cost quickly. 

## ⚠️ Important: Update Paths  

Before running any scripts, **ensure that all paths are updated to match your local environment**.  
The scripts and configurations may contain hardcoded paths that need to be adjusted for your system.

## 🌍 Infrastructure Setup

This project utilizes multiple **Vultr VPS instances** to distribute scanning efforts and avoid detection from using a single IP address. You will also need to setup your own .env files with Webhooks accordingly.

![image](https://github.com/user-attachments/assets/20fa4f39-07d9-4764-8e6e-8ea949188b69)


### 💾 Data Synchronization  

- A **PostgreSQL database** is used to store collected scan results and information.(only for the ad-hoc subdomain-exploting.go tool)
- Multiple **Vultr VPS instances** are set up, each responsible for scanning different targets.
- A **primary VPS** acts as the central repository then **rsync** is used to copy all the data from the main VPS instance every night, to all other available VPS's. The idea being that you can have three running or ten, the choice is yours. This helps consolidate data without performing scans from a single IP address, which could lead to IP ban. 
- The whole setup was designed to be **cost-effective**, maintaining continuous scanning for around **$100/month** but with a sudo version of IP rotation.

### 🔄 Automated Data Collection  

Each night, the system fetches target data from multiple sources, including:

- **HackerOne**
- **Bugcrowd**
- **Intigriti**
- **YesWeHack**

This is achieved via their respective APIs and updated into the database. The relevant domains are pulled and stored in a file. 

- At the time of writing this my system pulls from an updated list each morning from around 8000 root domains.

📋Future plans; 
🐋Start looking at K8's and making this a containered tool
🚛Add queues and make the results from massdns.go > into the database.
🚪Spend some more time on the Open-Redirect and XSS tools. 
