# Deploy via Docker Swarm and Cloudflare

This deployment guide assumes you are using Docker with Swarm mode using Traefik as the ingress and CLoudflare for DNS and Reverse Proxying (Optional).

There are many other ways to deploy this which we will cover eventually, but for now Docker Swarm is fairly simple.

> [!NOTE]  
> The use of Cloudflare is entirely optional in this setup,
> you _could_ swap it for any DNS provider.
> It is used by Traefik to automatically configure DNS-01 challenges for
> Let's Encrypt for TLS certificates.

## Requirements

- A VPS or VM publicly accessible
- A domain name
- DNS hosted on Cloudflare
- A cup of coffee

## Setup

- Update and configure your VPS/VM how you prefer.  Hardening and configuration is out of scope for this guide
- Ensure ports 80/443(tcp) are open via IPTables or UFW, however you choose to open them
- Visit cloudflare and point `domain.tld` to the public IP address of your system, ensure proxy is checked
- Grab your API key from the Cloudflare interface
- Install docker on your VPS/VM
- Create an operator user with `wheel` or `sudo` group as well as `docker` group
- Execute: `docker swarm init`, you've now created a single node Docker Swarm Cluster
- Execute: `docker network create -d overlay traefik` this will create the network needed for Traefik and Yarn to communicate
- Execute: `docker stack deploy -c traefik.yml traefik` this will deploy the traefik stack
- Give Traefik a few minutes to warm up.  Tail the logs with `docker logs -f traefik_traefik.1.someid` to ensure there are no errors
- Execute: `docker stack deploy -c yarn.yml yarn`
- Execute `watch docker stack ps yarn --no-trunc` to ensure the stack comes up and reaches running state
- Execute: `docker logs -f yarn_yarn.1.someid` to verify the store is merged and service is up
- If there are no issues, you should be able to visit: `https://mypoddomain.com`
- To create your admin user, register with the username and email you set in the `yarn.yml` file
- Login to your pod to verify it was successful
- You can now customise your pod by visiting Settings -> Poderator Settings
- Grab another coffee