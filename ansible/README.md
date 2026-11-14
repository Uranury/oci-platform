# tsis1linux: Ansible

One playbook, `site.yml`, takes two plain Ubuntu 24.04 ARM64 VMs on OCI to the
current state of `mainapp` (nginx + Go API) and `main-db` (PostgreSQL 16 +
backups), then runs smoke tests. It replaces the scripts in `../deploy/` and
`../local/scripts/`, plus the manual steps from the SIS #1–#3 reports.

## Layout

```
ansible/
  ansible.cfg                 inventory path, roles_path, pipelining, ssh options
  site.yml                    single entry point
  requirements.yml            collections: ansible.posix, community.postgresql, community.general
  inventory/
    hosts.yml                 groups app (mainapp) and db (main-db, via mainapp)
    group_vars/all/vars.yml   accounts, groups, app name, private IPs, timezone
    group_vars/all/vault.yml  ENCRYPTED: vault_db_password, vault_jwt_secret
    group_vars/app.yml        firewall rules, package versions, release retention
    group_vars/db.yml         firewall rule, listen_addresses, backup settings
  files/keys/*.pub            public halves only: admin (per machine) and deploy bot
  roles/                      one role per piece of the old setup (table below)
  scripts/check-idempotence.sh
```

`group_vars` sits next to the inventory (not next to `site.yml`) so that ad-hoc
commands such as `ansible all -m ping` see the same variables as the playbook.

| Role | Hosts | Replaces |
|---|---|---|
| `base` | all | SIS #1 checklist: upgrade, hostname, timezone, `ncurses-term`, `acl`; SIS #3: purge `rpcbind`/`nfs-common` |
| `users` | all | `setup-users.sh` |
| `sudoers` | all | `setup-sudoers.sh` (`visudo -cf` before install) |
| `ssh` | all | `setup-ssh.sh key` / `harden` (`sshd -t` before install) |
| `firewall` | all | `setup-firewall.sh app\|db` + `confirm` (`nft -c`, 3-minute rollback timer) |
| `packages` | app | `setup-packages.sh` (jq/netcat, PGDG client 17, hurl 8.0.1 by checksum) |
| `postgresql` | db | SIS #1 report 3.5 (manual) |
| `backup` | db | `setup-fs.sh db` |
| `nginx` | app | manual install + site from `setup-fs.sh app` (`nginx -t` before reload) |
| `app` | app | `setup-fs.sh app` + `enable-service.sh` |
| `deploy_release` | app | `local/scripts/push-release.sh` + `activate-release.sh` (+ keep last 5) |
| `acl` | all | `setup-perms.sh` (runs last in each play) |
| `bootstrap_cleanup` | all | SIS #3 §6: remove the `ubuntu` login |
| `verify` | all + controller | `verify.sh` + `local/scripts/smoke.sh` |

## Requirements on the laptop

- Ansible (ansible-core 2.16+) and the collections:
  `ansible-galaxy collection install -r requirements.yml`
- Go (the playbook builds the linux/arm64 binary itself), git
- The private keys referenced in `inventory/hosts.yml`:
  `~/.ssh/mainapp_keys` (mainapp), `~/.ssh/maindbprivate.key` (main-db),
  optionally `~/.ssh/tsis1-deploy` (only for the bot login check)
- The vault password: `--ask-vault-pass`, or put it in `ansible/.vault_pass`
  (gitignored, mode 600) and use `--vault-password-file .vault_pass`

## Variables to set

| Where | Variable | Meaning |
|---|---|---|
| `inventory/hosts.yml` | `app_public_ip` | mainapp's public IP |
| `inventory/hosts.yml` | `main-db.ansible_host` | main-db's private IP |
| `inventory/hosts.yml` | `app_ssh_key`, `db_ssh_key` | private keys chosen for the VMs in the OCI console |
| `group_vars/all/vars.yml` | `app_private_ip`, `db_private_ip` | private IPs (firewall, pg_hba, DATABASE_URL) |
| `group_vars/all/vars.yml` | `remove_bootstrap_user` | `true` (default) removes `ubuntu` as in SIS #3; `false` keeps it as break-glass |
| `group_vars/all/vault.yml` | `vault_db_password`, `vault_jwt_secret` | `ansible-vault edit inventory/group_vars/all/vault.yml` |
| `-e` | `bootstrap_user=ubuntu` | first run only: log in as the cloud image user |
| `-e` | `apt_online=true` | while main-db has its temporary public IP |
| `-e` | `deploy_release_binary=/path/tsis1linux` | deploy a prebuilt binary (e.g. the CI artifact) instead of building |

## Bootstrapping new VMs

1. In the OCI console create two VMs from the default Canonical Ubuntu 24.04
   image (shape `VM.Standard.A1.Flex`, same VCN and subnet). Upload
   `~/.ssh/mainapp_keys.pub` for the app VM and the main-db public key for the
   db VM. Attach NSG `devops-tokyNSG` to the app VM and `db-nsg` to the db VM.
   Give the db VM a **temporary** public IP (it needs the Ubuntu archive once).
2. Put the new IPs in `inventory/hosts.yml` (and `group_vars/all/vars.yml` if
   the private IPs changed). If an IP was used before, drop its old host key:
   `ssh-keygen -R <ip>`.
3. First run, as `ubuntu`, with internet on main-db:

   ```
   cd ansible
   ansible-galaxy collection install -r requirements.yml
   ansible-playbook site.yml --ask-vault-pass -e bootstrap_user=ubuntu -e apt_online=true
   ```

   On the way the playbook creates `alibi`, checks that `alibi` can log in,
   hardens sshd, applies the firewall, installs everything, deploys the
   current commit, then logs in as `alibi` and deletes the `ubuntu` account.
4. Remove the temporary public IP from main-db in the OCI console.
5. Every later run (as `alibi`) must change nothing:

   ```
   scripts/check-idempotence.sh --ask-vault-pass
   ```

   Run it from the same commit as step 3: a new commit is a new release.
6. Reboot test: reboot both VMs, then run only the tests:

   ```
   ansible-playbook site.yml --ask-vault-pass --tags verify
   ```

## Everyday commands

```
ansible-playbook site.yml --ask-vault-pass --tags verify       # smoke tests only
ansible-playbook site.yml --ask-vault-pass --tags deploy       # build + deploy the current commit
ansible-playbook site.yml --ask-vault-pass --check --diff      # dry run
ansible-playbook site.yml --syntax-check && ansible-lint site.yml
```

## Design notes

- **Bootstrap user.** `ansible_user` is `{{ connect_user }}`, which is
  `bootstrap_user` if given, otherwise `alibi`. Tasks that must prove the admin
  login (in `ssh`, `bootstrap_cleanup` and `verify`) override `connect_user`.
- **main-db through mainapp.** `ansible_ssh_common_args` uses
  `ProxyCommand="ssh -W %h:%p -i <app key> …"`, the long form of ProxyJump:
  `-o ProxyJump` cannot be given the jump host's key on the command line.
- **Firewall lockout protection.** The ruleset is checked with `nft -c` first.
  Before it is loaded, a systemd timer is armed to delete the table after 3
  minutes. Ansible then drops its connection (`meta: reset_connection`) and
  must open a new one (`wait_for_connection`) before the timer is cancelled.
  The OCI security list and NSGs remain the outer layer.
- **ACLs last.** Every module that rewrites a file replaces its inode and loses
  its ACL, so `acl` runs after `sudoers` and `postgresql`.
- **Release source.** By default the playbook builds the binary on the laptop
  (`CGO_ENABLED=0 GOOS=linux GOARCH=arm64`, `-trimpath`, so the same commit gives
  the same bytes) and names the release `git describe --always --dirty`. To
  deploy the GitHub Actions artifact instead:
  `gh run download <run-id> -n tsis1linux-<sha> -D /tmp/rel` and
  `-e deploy_release_binary=/tmp/rel/tsis1linux -e deploy_release_version=<sha>`.
- **Out of scope.** The VMs, VCN, security list and NSGs are created by hand in
  the OCI console (no `oracle.oci` collection).
