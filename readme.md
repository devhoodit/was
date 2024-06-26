![was_image](./docs/was.png)

<div align="center">
<h1>Modoo collection WAS</h1>
</div>

Modoo collection web application server

- [configuration](#configuration)
- [how to run](#how-to-run)
- [build](#build)
- [security](#security)
- [test](#test-and-develop)
- [background service (docker compose)](#background-service-mysql-and-redis)

## Configuration

Ref [example.yaml](./example.yaml), rename to config.yaml

### database

One of following options

- sqlite
  |Name|value|property|
  |---|---|---|
  |path|tmp/sqlite.db|db file path|

  This option for using SQLite

- mysql
  |Name|value|property|
  |---|---|---|
  |username|root|username|
  |password|password|password|
  |address|127.0.0.1|db address|
  |port|3306|db port|
  |maxIdelConns|10|db max idle connections|
  |maxOpenConns|100|db max open connections|
  |maxLifeTime|180|connection max life time|

  This option for using MySQL

### email

One of following options

- mock
  |Name|value|property|
  |---|---|---|
  |type|cli|config how to process when email verify|

  If "cli", print on command line, else, nothing  
  This option for testing other functions

- ses
  |Name|value|property|
  |---|---|---|
  |region|us-east-2|aws ses region|
  |domain|your_domain.com|aws route53 domain|
  |id (Optional)|secret_id|aws s3 access id|
  |key (Optional)|secret_key|aws s3 access key|

  id and key pair is optional, if there is no id and key value, then server will retrieve aws ec2 temporary credential

### storage

One of following options

- local
  |Name|value|property|
  |---|---|---|
  |baseDirectory|/root/server/tmpstorage|directory where server static files will be stored like, image etc.|

- s3
  |Name|value|property|
  |---|---|---|
  |region|us-east-2|aws s3 region|
  |bucketName|bucket0|aws s3 bucket name|
  |id (Optional)|secret_id|aws s3 access id|
  |key (Optional)|secret_key|aws s3 access key|

  id and key pair is optional, if there is no id and key value, then server will retrieve aws ec2 temporary credential

## How to run

Ref [example.yaml](./example.yaml), rename to config.yaml  
If you want to use docker network, [see this](https://docs.docker.com/network/)

```console
make docker-build
make docker-image
docker run -d -p 443:443 -v "${PWD}/logs:/server/logs" -v "${PWD}/config.yaml:/server/config.yaml" --network backnet was
```

## Build

### Build (Native)

Require go

```console
# build default (linux-amd64)
make build # or make build-linux-amd64

# build all
make build-all
```

#### Options

- build-linux-amd64 (build-default)
- build-linux-arm64
- build-windows-amd64
- build-windows-arm64
- build-all

### Build (Docker)

```console
# build default (linux-amd64)
make docker-build # or make docker-build-linux-amd64

# build all
make docker-build-all
```

#### Options

- docker-build-linux-amd64 (docker-build-default)
- docker-build-linux-arm64
- docker-build-windows-amd64
- docker-build-windows-arm64
- docker-build-all

### Build Docker Image

With current ./config.yaml

```console
# build docker default image (linux-amd64)
make docker-image # or make docker-image-linux-amd64
# tag was

#build linux-arm64 docker image
make docker-image-linux-arm64
# tag arm64/was
```

## Security
> [!WARNING]
> This section explain how to setup secure server structure

1. Database access   
Dzo not use root account to access database, create new user with restricted access range (private network only), grant only specific database  

2. AWS configuration  
In SES or S3 configuration, id and key configuration is not recommended  
Modoo Collection WAS support [Temporary security credentials in IAM](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html)  
Automatically detect EC2 Roles and provide temporary security credentials for accessing AWS resources  
If you need more information about this, [follow this link](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_use_switch-role-ec2.html)  
We highly recommand that use this option (if you leave id and key blank, automatically use this option, See [Configuration](#configuration))  

Database access management  
Do not use root account to access database, make new role with restricted access

## Test and Develop

> [!WARNING]
> Do not use test setting in production!

## Background service (mysql and redis)

Redis is required (MySQL optional, SQLite)  
Ref [./compose.yaml](./compose.yaml)

```console
docker compose up -d
```

> [!CAUTION]
> Do not expose your PORT, ./compose.yaml expose ports
> In production, Please remove PORT option