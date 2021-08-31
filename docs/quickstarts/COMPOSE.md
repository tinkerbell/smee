# Quick start guide for Docker Compose

This option will stand up the provisioner using Docker Compose.
You will need to bring your own machines to provision.

## Prerequisites

- [Git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) is installed
- [Docker](https://docs.docker.com/get-docker/) is installed (version >= 19.03)
- [Docker Compose](https://docs.docker.com/compose/install/) is installed (version >= 1.29.0)

## Steps

1. Clone this repository

   ```bash
   git clone https://github.com/tinkerbell/boots.git
   cd boots
   ```

2. Set the public IP address for the provisioner

   ```bash
   # This should be an IP that's on an interface where you will be provisioning machines
   export TINKERBELL_HOST_IP=192.168.2.111
   ```

3. Set the IP and MAC address of the machine you want to provision (if you want Tink hardware, template, and workflow records auto-generated)

   ```bash
   # This IP and MAC of the machine to be provisioned
   # The IP should normally be in the same network as the IP used for the provisioner
   export TINKERBELL_CLIENT_IP=192.168.2.211
   export TINKERBELL_CLIENT_MAC=08:00:27:9E:F5:3A
   ```

4. Start the provisioner

   ```bash
   cd deploy/stack
   docker-compose up -d
   # This process will take about 5-10 minutes depending on your internet connection.
   # OSIE is about 2GB in size and the Ubuntu Focal image is about 500MB
   ```

   <details>
   <summary>expected output</summary>

   ```bash
   Creating network "deploy_default" with the default driver
   Creating volume "deploy_postgres_data" with default driver
   Creating volume "deploy_certs" with default driver
   Creating volume "deploy_auth" with default driver
   Pulling tls-gen (cfssl/cfssl:)...
   latest: Pulling from cfssl/cfssl
   50e431f79093: Pull complete
   dd8c6d374ea5: Pull complete
   c85513200d84: Pull complete
   55769680e827: Pull complete
   15357f5e50c4: Pull complete
   5dcdacb923a0: Pull complete
   3b6795100674: Pull complete
   572d1ec44610: Pull complete
   b3de5480e786: Pull complete
   03ed00a20dd4: Pull complete
   Digest: sha256:c21e852f3904e2ba77960e9cba23c69d9231467795a8a160ce1d848e621381ea
   Status: Downloaded newer image for cfssl/cfssl:latest
   Pulling registry-auth (httpd:2)...
   2: Pulling from library/httpd
   b4d181a07f80: Pull complete
   4b72f5187e6e: Pull complete
   12b2c44d04b2: Pull complete
   35c238b46d30: Pull complete
   1adcec05f52b: Pull complete
   Digest: sha256:1fd07d599a519b594b756d2e4e43a72edf7e30542ce646f5eb3328cf3b12341a
   Status: Downloaded newer image for httpd:2
   Pulling osie-work (alpine:)...
   latest: Pulling from library/alpine
   5843afab3874: Pull complete
   Digest: sha256:234cb88d3020898631af0ccbbcca9a66ae7306ecd30c9720690858c1b007d2a0
   Status: Downloaded newer image for alpine:latest
   Pulling ubuntu-image-setup (ubuntu:)...
   latest: Pulling from library/ubuntu
   a31c7b29f4ad: Pull complete
   Digest: sha256:b3e2e47d016c08b3396b5ebe06ab0b711c34e7f37b98c9d37abe794b71cea0a2
   Status: Downloaded newer image for ubuntu:latest
   Pulling db (postgres:10-alpine)...
   10-alpine: Pulling from library/postgres
   5843afab3874: Already exists
   525703b16f79: Pull complete
   86f8340bd3d9: Pull complete
   79044baaabb2: Pull complete
   36cf25109e96: Pull complete
   bc752cda1992: Pull complete
   17905079c3e2: Pull complete
   04d52afe0744: Pull complete
   325e49088bb1: Pull complete
   Digest: sha256:0eef1c94e0c4b0c4b84437785d0c5926f62b7f537627d97cf9ebcd7b205bc9aa
   Status: Downloaded newer image for postgres:10-alpine
   Pulling tink-server-migration (quay.io/tinkerbell/tink:sha-8ea8a0e5)...
   sha-8ea8a0e5: Pulling from tinkerbell/tink
   ddad3d7c1e96: Pull complete
   721d7c3819bb: Pull complete
   bb168258b5ea: Pull complete
   Digest: sha256:84fc83f8562901d0b27e7ebb453a7f27e5797d17fb0b6899f92002df840fbf21
   Status: Downloaded newer image for quay.io/tinkerbell/tink:sha-8ea8a0e5
   Pulling create-tink-records (quay.io/tinkerbell/tink-cli:sha-8ea8a0e5)...
   sha-8ea8a0e5: Pulling from tinkerbell/tink-cli
   ddad3d7c1e96: Already exists
   30e0616e2b41: Pull complete
   5066d7cb6405: Pull complete
   Digest: sha256:0fc5441e9ef6e94eff7bf1ae9cf9a15a98581c742890d2d7130fd9542b12802d
   Status: Downloaded newer image for quay.io/tinkerbell/tink-cli:sha-8ea8a0e5
   Pulling registry (registry:2.7.1)...
   2.7.1: Pulling from library/registry
   ddad3d7c1e96: Already exists
   6eda6749503f: Pull complete
   363ab70c2143: Pull complete
   5b94580856e6: Pull complete
   12008541203a: Pull complete
   Digest: sha256:aba2bfe9f0cff1ac0618ec4a54bfefb2e685bbac67c8ebaf3b6405929b3e616f
   Status: Downloaded newer image for registry:2.7.1
   Pulling images-to-local-registry (quay.io/containers/skopeo:latest)...
   latest: Pulling from containers/skopeo
   7d63354ae1f4: Pull complete
   75b596804be5: Pull complete
   51cba67b7076: Pull complete
   18e224798580: Pull complete
   2b087916826e: Pull complete
   695aa9f61886: Pull complete
   Digest: sha256:ab8d8b9e7f61b78a7f24ec4076e0ef895e596cf31dd32d9ee6e718c571f02cde
   Status: Downloaded newer image for quay.io/containers/skopeo:latest
   Pulling boots (quay.io/tinkerbell/boots:sha-cb0290f8)...
   sha-cb0290f8: Pulling from tinkerbell/boots
   339de151aab4: Pull complete
   edc0420940c8: Pull complete
   85dd670aacee: Pull complete
   Digest: sha256:8e106bf73122d08ce9ef75f5cae4be77ecff38c2b55cb44541caabf94d325de9
   Status: Downloaded newer image for quay.io/tinkerbell/boots:sha-cb0290f8
   Pulling osie-bootloader (nginx:alpine)...
   alpine: Pulling from library/nginx
   5843afab3874: Already exists
   0dc18a5274f2: Pull complete
   48a0ee941dcd: Pull complete
   2446243a1a3f: Pull complete
   cbf0756b41fb: Pull complete
   c72750a979b9: Pull complete
   Digest: sha256:91528597e842ab1b3b25567191fa7d4e211cb3cc332071fa031cfed2b5892f9e
   Status: Downloaded newer image for nginx:alpine
   Pulling hegel (quay.io/tinkerbell/hegel:sha-9f5da0a8)...
   sha-9f5da0a8: Pulling from tinkerbell/hegel
   5d20c808ce19: Pull complete
   c9b8cebd2f86: Pull complete
   a3e87bc13599: Pull complete
   ac61d4bd540b: Pull complete
   Digest: sha256:9d3c6d5e4bc957cedafbeec22da4f59d94c78b65d84adbd0c8f947c51cf3668b
   Status: Downloaded newer image for quay.io/tinkerbell/hegel:sha-9f5da0a8
   Creating deploy_registry-auth_1      ... done
   Creating deploy_tls-gen_1            ... done
   Creating deploy_db_1                 ... done
   Creating deploy_osie-work_1          ... done
   Creating deploy_ubuntu-image-setup_1 ... done
   Creating deploy_osie-bootloader_1    ... done
   Creating deploy_registry_1           ... done
   Creating deploy_tink-server-migration_1 ... done
   Creating deploy_tink-server_1           ... done
   Creating deploy_images-to-local-registry_1 ... done
   Creating deploy_create-tink-records_1      ... done
   Creating deploy_hegel_1                    ... done
   Creating deploy_registry-ca-crt-download_1 ... done
   Creating deploy_boots_1                    ... done
   Creating deploy_tink-cli_1                 ... done
   ```

   </details>

5. Power up the machine to be provisioned

6. Watch for the provisioner to complete

   ```bash
   # watch the workflow events and status for workflow completion
   # once the workflow is complete (see the expected output below for completion), move on to the next step
   wid=$(cat deploy/stack/compose/manifests/workflow/workflow_id.txt); docker exec -it stack_tink-cli_1 watch "tink workflow events ${wid}; tink workflow state ${wid}"
   ```

   <details>
     <summary>expected output</summary>

   ```bash
   +--------------------------------------+-----------------+---------------------+----------------+---------------------------------+---------------+
   | WORKER ID                            | TASK NAME       | ACTION NAME         | EXECUTION TIME | MESSAGE                         | ACTION STATUS |
   +--------------------------------------+-----------------+---------------------+----------------+---------------------------------+---------------+
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | stream-ubuntu-image |              0 | Started execution               | STATE_RUNNING |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | stream-ubuntu-image |             15 | finished execution successfully | STATE_SUCCESS |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | install-openssl     |              0 | Started execution               | STATE_RUNNING |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | install-openssl     |              1 | finished execution successfully | STATE_SUCCESS |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | create-user         |              0 | Started execution               | STATE_RUNNING |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | create-user         |              0 | finished execution successfully | STATE_SUCCESS |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | enable-ssh          |              0 | Started execution               | STATE_RUNNING |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | enable-ssh          |              0 | finished execution successfully | STATE_SUCCESS |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | disable-apparmor    |              0 | Started execution               | STATE_RUNNING |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | disable-apparmor    |              0 | finished execution successfully | STATE_SUCCESS |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | write-netplan       |              0 | Started execution               | STATE_RUNNING |
   | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 | os-installation | write-netplan       |              0 | finished execution successfully | STATE_SUCCESS |
   +--------------------------------------+-----------------+---------------------+----------------+---------------------------------+---------------+
   +----------------------+--------------------------------------+
   | FIELD NAME           | VALUES                               |
   +----------------------+--------------------------------------+
   | Workflow ID          | 3107919b-e59d-11eb-bf99-0242ac120005 |
   | Workflow Progress    | 100%                                 |
   | Current Task         | os-installation                      |
   | Current Action       | write-netplan                        |
   | Current Worker       | 0eba0bf8-3772-4b4a-ab9f-6ebe93b90a94 |
   | Current Action State | STATE_SUCCESS                        |
   +----------------------+--------------------------------------+
   ```

   </details>

7. Reboot the machine

8. Login to the machine

   The machine has been provisioned with Ubuntu Focal.
   You can now SSH into the machine.

   ```bash
   # crtl-c to exit the watch
   ssh tink@${TINKERBELL_CLIENT_IP} # user/pass => tink/tink
   ```
