# Quick start guide for Vagrant and VirtualBox

This option will stand up the provisioner in Virtualbox using Vagrant.
This option will also show you how to create a machine to provision.

## Prerequisites

- [Vagrant](https://www.vagrantup.com/downloads) is installed
- [VirtualBox](https://www.virtualbox.org/) is installed

## Steps

1. Clone this repository

   ```bash
   git clone https://github.com/tinkerbell/boots.git
   cd boots
   ```

2. Start the provisioner

   ```bash
   cd deploy/stack/vagrant
   vagrant up
   # This process will take about 5-10 minutes depending on your internet connection.
   # OSIE is about 2GB in size and the Ubuntu Focal image is about 500MB
   ```

   <details>
   <summary>expected output</summary>

   ```bash
   Bringing machine 'provisioner' up with 'virtualbox' provider...
   ==> provisioner: Importing base box 'generic/ubuntu2004'...
   ==> provisioner: Matching MAC address for NAT networking...
   ==> provisioner: Checking if box 'generic/ubuntu2004' version '3.2.24' is up to date...
   ==> provisioner: Setting the name of the VM: vagrant_provisioner_1626366679197_92753
   ==> provisioner: Clearing any previously set network interfaces...
   ==> provisioner: Preparing network interfaces based on configuration...
       provisioner: Adapter 1: nat
       provisioner: Adapter 2: hostonly
   ==> provisioner: Forwarding ports...
       provisioner: 22 (guest) => 2222 (host) (adapter 1)
   ==> provisioner: Running 'pre-boot' VM customizations...
   ==> provisioner: Booting VM...
   ==> provisioner: Waiting for machine to boot. This may take a few minutes...
       provisioner: SSH address: 127.0.0.1:2222
       provisioner: SSH username: vagrant
       provisioner: SSH auth method: private key
       provisioner:
       provisioner: Vagrant insecure key detected. Vagrant will automatically replace
       provisioner: this with a newly generated keypair for better security.
       provisioner:
       provisioner: Inserting generated public key within guest...
       provisioner: Removing insecure key from the guest if it's present...
       provisioner: Key inserted! Disconnecting and reconnecting using new SSH key...
   ==> provisioner: Machine booted and ready!
   ==> provisioner: Checking for guest additions in VM...
   ==> provisioner: Configuring and enabling network interfaces...
   ==> provisioner: Mounting shared folders...
       provisioner: /vagrant => /Users/jacobweinstock/tmp/sandbox/deploy
   ==> provisioner: Running provisioner: docker...
       provisioner: Installing Docker onto machine...
   ==> provisioner: Running provisioner: docker_compose...
       provisioner: Checking for Docker Compose installation...
       provisioner: Getting machine and kernel name from guest machine...
       provisioner: Downloading Docker Compose 1.29.1 for Linux x86_64
       provisioner: Downloaded Docker Compose 1.29.1 has SHA256 signature 8097769d32e34314125847333593c8edb0dfc4a5b350e4839bef8c2fe8d09de7
       provisioner: Uploading Docker Compose 1.29.1 to guest machine...
       provisioner: Installing Docker Compose 1.29.1 in guest machine...
       provisioner: Symlinking Docker Compose 1.29.1 in guest machine...
       provisioner: Running docker-compose up...
   ==> provisioner: Creating network "vagrant_default" with the default driver
   ==> provisioner: Creating volume "vagrant_postgres_data" with default driver
   ==> provisioner: Creating volume "vagrant_certs" with default driver
   ==> provisioner: Creating volume "vagrant_auth" with default driver
   ==> provisioner: Pulling tls-gen (cfssl/cfssl:)...
       provisioner: latest: Pulling from cfssl/cfssl
       provisioner: Digest: sha256:c21e852f3904e2ba77960e9cba23c69d9231467795a8a160ce1d848e621381ea
       provisioner: Status: Downloaded newer image for cfssl/cfssl:latest
   ==> provisioner: Pulling registry-auth (httpd:2)...
       provisioner: 2: Pulling from library/httpd
       provisioner: Digest: sha256:1fd07d599a519b594b756d2e4e43a72edf7e30542ce646f5eb3328cf3b12341a
       provisioner: Status: Downloaded newer image for httpd:2
   ==> provisioner: Pulling osie-work (alpine:)...
       provisioner: latest: Pulling from library/alpine
       provisioner: Digest: sha256:234cb88d3020898631af0ccbbcca9a66ae7306ecd30c9720690858c1b007d2a0
       provisioner: Status: Downloaded newer image for alpine:latest
   ==> provisioner: Pulling ubuntu-image-setup (ubuntu:)...
       provisioner: latest: Pulling from library/ubuntu
       provisioner: Digest: sha256:b3e2e47d016c08b3396b5ebe06ab0b711c34e7f37b98c9d37abe794b71cea0a2
       provisioner: Status: Downloaded newer image for ubuntu:latest
   ==> provisioner: Pulling db (postgres:10-alpine)...
       provisioner: 10-alpine: Pulling from library/postgres
       provisioner: Digest: sha256:0eef1c94e0c4b0c4b84437785d0c5926f62b7f537627d97cf9ebcd7b205bc9aa
       provisioner: Status: Downloaded newer image for postgres:10-alpine
   ==> provisioner: Pulling tink-server-migration (quay.io/tinkerbell/tink:sha-8ea8a0e5)...
       provisioner: sha-8ea8a0e5: Pulling from tinkerbell/tink
       provisioner: Digest: sha256:84fc83f8562901d0b27e7ebb453a7f27e5797d17fb0b6899f92002df840fbf21
       provisioner: Status: Downloaded newer image for quay.io/tinkerbell/tink:sha-8ea8a0e5
   ==> provisioner: Pulling create-tink-records (quay.io/tinkerbell/tink-cli:sha-8ea8a0e5)...
       provisioner: sha-8ea8a0e5: Pulling from tinkerbell/tink-cli
       provisioner: Digest: sha256:0fc5441e9ef6e94eff7bf1ae9cf9a15a98581c742890d2d7130fd9542b12802d
       provisioner: Status: Downloaded newer image for quay.io/tinkerbell/tink-cli:sha-8ea8a0e5
   ==> provisioner: Pulling registry (registry:2.7.1)...
       provisioner: 2.7.1: Pulling from library/registry
       provisioner: Digest: sha256:aba2bfe9f0cff1ac0618ec4a54bfefb2e685bbac67c8ebaf3b6405929b3e616f
       provisioner: Status: Downloaded newer image for registry:2.7.1
   ==> provisioner: Pulling images-to-local-registry (quay.io/containers/skopeo:latest)...
       provisioner: latest: Pulling from containers/skopeo
       provisioner: Digest: sha256:f7bfc49ffc4331ce7ab6ff51b0883bc39115cd1028fe1606a6fc9d4351df3673
       provisioner: Status: Downloaded newer image for quay.io/containers/skopeo:latest
   ==> provisioner: Pulling boots (quay.io/tinkerbell/boots:sha-cb0290f8)...
       provisioner: sha-cb0290f8: Pulling from tinkerbell/boots
       provisioner: Digest: sha256:8e106bf73122d08ce9ef75f5cae4be77ecff38c2b55cb44541caabf94d325de9
       provisioner: Status: Downloaded newer image for quay.io/tinkerbell/boots:sha-cb0290f8
   ==> provisioner: Pulling osie-bootloader (nginx:alpine)...
       provisioner: alpine: Pulling from library/nginx
       provisioner: Digest: sha256:91528597e842ab1b3b25567191fa7d4e211cb3cc332071fa031cfed2b5892f9e
       provisioner: Status: Downloaded newer image for nginx:alpine
   ==> provisioner: Pulling hegel (quay.io/tinkerbell/hegel:sha-9f5da0a8)...
       provisioner: sha-9f5da0a8: Pulling from tinkerbell/hegel
       provisioner: Digest: sha256:9d3c6d5e4bc957cedafbeec22da4f59d94c78b65d84adbd0c8f947c51cf3668b
       provisioner: Status: Downloaded newer image for quay.io/tinkerbell/hegel:sha-9f5da0a8
   ==> provisioner: Creating vagrant_db_1 ...
   ==> provisioner: Creating vagrant_osie-bootloader_1 ...
   ==> provisioner: Creating vagrant_ubuntu-image-setup_1 ...
   ==> provisioner: Creating vagrant_tls-gen_1            ...
   ==> provisioner: Creating vagrant_registry-auth_1      ...
   ==> provisioner: Creating vagrant_osie-work_1          ...
   ==> provisioner: Creating vagrant_tls-gen_1            ... done
   ==> provisioner: Creating vagrant_ubuntu-image-setup_1 ... done
   ==> provisioner: Creating vagrant_osie-bootloader_1    ... done
   ==> provisioner: Creating vagrant_registry-auth_1      ... done
   ==> provisioner: Creating vagrant_db_1                 ... done
   ==> provisioner: Creating vagrant_osie-work_1          ... done
   ==> provisioner: Creating vagrant_registry_1           ...
   ==> provisioner: Creating vagrant_registry_1           ... done
   ==> provisioner: Creating vagrant_tink-server-migration_1 ...
   ==> provisioner: Creating vagrant_tink-server-migration_1 ... done
   ==> provisioner: Creating vagrant_tink-server_1           ...
   ==> provisioner: Creating vagrant_tink-server_1           ... done
   ==> provisioner: Creating vagrant_images-to-local-registry_1 ...
   ==> provisioner: Creating vagrant_images-to-local-registry_1 ... done
   ==> provisioner: Creating vagrant_registry-ca-crt-download_1 ...
   ==> provisioner: Creating vagrant_create-tink-records_1      ...
   ==> provisioner: Creating vagrant_boots_1                    ...
   ==> provisioner: Creating vagrant_tink-cli_1                 ...
   ==> provisioner: Creating vagrant_hegel_1                    ...
   ==> provisioner: Creating vagrant_boots_1                    ... done
   ==> provisioner: Creating vagrant_create-tink-records_1      ... done
   ==> provisioner: Creating vagrant_tink-cli_1                 ... done
   ==> provisioner: Creating vagrant_registry-ca-crt-download_1 ... done
   ==> provisioner: Creating vagrant_hegel_1                    ... done
   ==> provisioner: Creating vagrant_wait-for-osie-and-ubuntu-downloads_1 ...
   ==> provisioner: Creating vagrant_wait-for-osie-and-ubuntu-downloads_1 ... done
   ```

   </details>

3. Start the machine to be provisioned

   ```bash
   vagrant up machine1
   # This will start a VM to pxe boot. A GUI window of this machines console will be opened.
   # The `vagrant up machine1` command will exit quickly and show the following error message. This is expected.
   # Once the command line control is returned to you, you can move on to the next step.
   ```

   <details>
   <summary>expected output</summary>

   ```bash
   Bringing machine 'machine1' up with 'virtualbox' provider...
   ==> machine1: Importing base box 'jtyr/pxe'...
   ==> machine1: Matching MAC address for NAT networking...
   ==> machine1: Checking if box 'jtyr/pxe' version '2' is up to date...
   ==> machine1: Setting the name of the VM: vagrant_machine1_1626365105119_9800
   ==> machine1: Fixed port collision for 22 => 2222. Now on port 2200.
   ==> machine1: Clearing any previously set network interfaces...
   ==> machine1: Preparing network interfaces based on configuration...
       machine1: Adapter 1: hostonly
   ==> machine1: Forwarding ports...
       machine1: 22 (guest) => 2200 (host) (adapter 1)
       machine1: VirtualBox adapter #1 not configured as "NAT". Skipping port
       machine1: forwards on this adapter.
   ==> machine1: Running 'pre-boot' VM customizations...
   ==> machine1: Booting VM...
   ==> machine1: Waiting for machine to boot. This may take a few minutes...
       machine1: SSH address: 127.0.0.1:22
       machine1: SSH username: vagrant
       machine1: SSH auth method: private key
       machine1: Warning: Authentication failure. Retrying...
   Timed out while waiting for the machine to boot. This means that
   Vagrant was unable to communicate with the guest machine within
   the configured ("config.vm.boot_timeout" value) time period.

   If you look above, you should be able to see the error(s) that
   Vagrant had when attempting to connect to the machine. These errors
   are usually good hints as to what may be wrong.

   If you're using a custom box, make sure that networking is properly
   working and you're able to connect to the machine. It is a common
   problem that networking isn't setup properly in these boxes.
   Verify that authentication configurations are also setup properly,
   as well.

   If the box appears to be booting properly, you may want to increase
   the timeout ("config.vm.boot_timeout") value.

   ```

   </details>

4. Watch the provision complete

   ```bash
   # log in to the provisioner
   vagrant ssh provisioner
   # watch the workflow events and status for workflow completion
   # once the workflow is complete (see the expected output below for completion), move on to the next step
   wid=$(cat /vagrant/deploy/stack/compose/manifests/workflow/workflow_id.txt); docker exec -it stack_tink-cli_1 watch "tink workflow events ${wid}; tink workflow state ${wid}"
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

5. Reboot the machine

   ```bash
   # crtl-c to exit the watch
   # exit the provisioner
   vagrant@ubuntu2004:~$ exit
   # restart machine1
   # the output will be the same as step 3, once the command line control is returned to you, you can move on to the next step.
   vagrant reload machine1
   ```

6. Login to the machine

   The machine has been provisioned with Ubuntu Focal.
   You can now SSH into the machine.

   ```bash
   vagrant ssh provisioner
   ssh tink@192.168.50.43 # user/pass => tink/tink
   ```
