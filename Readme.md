# The Warwolf System

![The Warwolf, also known as ZonEdoNgAyeGo or WWF for short](warwolf.png)

Warwolf System is a simple, easy to use, easy to deploy and low maintenance required program that allows you to create private communication channels.

**In a nutshell, it's a proxy that uses plain old HTTP as internal transport and exposes a local Socks5 server as connection acceptor**

The application consists of two parts: A local server which will accept connections from your web browser or applications, and a backend server which will relay the connection.

You install the local server on your computer, deploy the backend server on an Internet VPS or the cloud where you can access it, then connect the two and you're all set.

Want a diagram?

                                            HTTP connection (Non Websocket)
                +----+             +----+        /
                |    | <-----------|<-->|--------------+       Somebody declared
                +----+             +----+              |          sovereignty
          Computer with the       Insecure       +====(|)====+     down here
       local server installed      factor       //     |     \\
                                               //   /--+--\   \+====================
    ===========N=E=T=W=O=R=K==C=L=I=F=F=======+/    |MAGIC|
                                                    \--+--/
           The destination                            /    \
                 host                                /    The backend server
                +----+                              /   (Sits on the Internet)
                |    | <---------------------------+
                +----+                         \
                                TCP connection or UDP packet route

Internally, the local and backend server will communicate using HTTP protocol, so it can work at pretty much anywhere.

Use case may include:

- When the public WIFI is insecure and you don't want to use a random VPN app because you _own_ a server on the public net.
- Your Internet is boringly fast and you want to slow it down so you live in the good old 80s again.
- ~~Your company only allows HTTP outgoings for employees and you want to get yourself fired~~ (Don't. How about just quit).
- ~~Your network admin only premits you with HTTP connections and you want to play UDP based games in front of him just to funk him up~~ (Don't. Should buy him cakes to bribe him for other connections).
- You already have HTTP services running on your server. You want to setup a proxy on that server but you don't want to expose any other port.
- _// TODO: Make up more buzzword use cases before publish project and delete this line_

## Features

- Supports Socks5 CONNECT command.
- Supports Socks5 UDP ASSOCIATE command as well.
- The backend HTTP transport is encrypted (even without HTTPS) via a shared key specified with `WWFKey` option. _(HTTPS is required if you want a really secured connection)_
- Very slow (2MBps max, or up to ~20Mbps if I'm your ISP).
- Maybe not include a nake picture of my cat.
- Oh and no any other third-party dependency than Go ~~(you know, that programming language which does not support generic)~~.

## Install and build

### I'm a pro

1. Then download this repository via `git clone`, `cd` to the project directory
2. Build this project with a `go` compiler (Only works on Go1)
3. The reset is self-explanatory, **_pro_**.

### Docker (recommended)

1. Install [Docker](https://docker.com)
2. Download this repository via `git clone`, then `cd` to the project directory
3. Run `docker build --tag wwf .`
4. The build proccess is really really fast, so I'm not going to prebuild anything for ya.

## Run

Yes, running is very good for your health. I run pretty much every day, 5 days a week at least. I love running ... away from my actual job, hehe. Also, I'm broke.

Anyway. After you got those code built, you'll found one and only one binary generated. Yes, the binary contains both the local and the backend server. To specify which role it will play, you need to setup an enviroment variable called `WWFAs`. Set it to `Server` and it will run as the backend server, set it to `Client` then it will become a local server. God, I really hope my kid can be this obedient, if I got any that is.

In fact, enviroment variables is how we will config the Warwolf System. All the options for both servers can be found in file [server.env](server.env) and [client.env](client.env).

For example, to run as the local server (assuming the generated binary is called `warwolf`):

    WWFAs=Client WWFBackend=https://google.appspot.com/fun-but-not-real WWFKey=DontTellAnyBody WWFListen=:2048 ./warwolf

To run as the backend server:

    WWFAs=Server WWFListen=:8080 WWFKey=DontTellAnyBody ./warwolf

If you want to be a _pro_, to run a local server:

    export WWFAs=Client
    export WWFBackend=https://microsoft.com/why-open-source-is-communism
    export WWFKey=ImNotAOneLiner
    export WWFListen=0.0.0.0:2048
    export WWFBackendHostEnforce=
    export WWFUsername=
    export WWFPassword=
    export WWFMaxClientConnections=256
    export WWFMaxBackendConnections=5
    export WWFMaxRetrieveLength=8192
    export WWFRequestTimeout=10
    export WWFIdleTimeout=30
    export WWFMaxRetries=16
    ./warwolf

And to run a backend server:

    export WWFAs=Server
    export WWFListen=:8080
    export WWFKey=ImNotAOneLiner
    export WWFIdleTimeout=60
    export WWFDialTimeout=5
    export WWFRetrieveTimeout=10
    export WWFMaxOutgoingConnections=256
    export WWFTLSPublicKeyBlock=
    export WWFTLSPrivateKeyBlock=
    ./warwolf

### But what about [Docker](https://docker.com)?

Well, use

    docker run --name wwf --detach --restart always \
      --net host \
      --env WWFAs=Client \
      --env WWFBackend=https://microsoft.com/why-open-source-is-communism \
      --env WWFKey=ImNotAOneLiner \
      --env WWFListen=0.0.0.0:1080 \
      --env WWFBackendHostEnforce= \
      --env WWFUsername= \
      --env WWFPassword= \
      --env WWFMaxClientConnections=256 \
      --env WWFMaxBackendConnections=5 \
      --env WWFMaxRetrieveLength=8192 \
      --env WWFRequestTimeout=10 \
      --env WWFIdleTimeout=30 \
      --env WWFMaxRetries=16 \
      wwf

for local server, or

    docker run --name wwf --detach --restart always --publish 8080:8080/tcp \
      --env WWFAs=Server \
      --env WWFListen=:8080 \
      --env WWFKey=ImNotAOneLiner \
      --env WWFIdleTimeout=60 \
      --env WWFDialTimeout=5 \
      --env WWFRetrieveTimeout=10 \
      --env WWFMaxOutgoingConnections=256 \
      --env WWFTLSPublicKeyBlock= \
      --env WWFTLSPrivateKeyBlock= \
      wwf

for backend server.

### Options explained

#### For the local server:

    WWFBackend=                     # The URL of the backend server
    WWFKey=                         # Shared key, must be the same on the server
    WWFListen=:1080                 # Listening port of the local Socks5 server
    WWFUsername=                    # Login user name of the local socks5 server
    WWFPassword=                    # Login password of the local socks5 server
    WWFBackendHostEnforce=          # Connect to this hostname rather than the one specified in the WWFBackend URL while keeping the request unchanged (Format: 127.0.0.1:443)
    WWFMaxClientConnections=256     # Max connections this client should sent out
    WWFMaxBackendConnections=5      # Max connections to be backend server
    WWFMaxRetrieveLength=8192       # How many bytes of data each retrieve can carry, set it lower when the connection is poor
    WWFRequestTimeout=10            # Max wait time for initial respond from the backend
    WWFIdleTimeout=30               # Max idle time for the backend connection
    WWFMaxRetries=16                # How many times to retry before given up the request

#### For the backend server:

    WWFListen=:8080                 # Listen port for the backend HTTP server
    WWFKey=                         # Shared key, must be the same on the client
    WWFIdleTimeout=60               # Max idle time for the outgoing connections
    WWFDialTimeout=5                # Max wait time for dialing to remote
    WWFRetrieveTimeout=10           # Max wait time for reading from remote
    WWFMaxOutgoingConnections=256   # Max remote connections
    WWFTLSPublicKeyBlock=           # Data of the certificate if you want to use TLS
    WWFTLSPrivateKeyBlock=          # Data of the certificate key if you want to use TLS

## Maintenance

Well as a hot-hearted member of _Low Maintenance International Elite Club (LMIeC)_, I've designed this software to be so low maintenance (Or _LowMain_ for short, as the opposite of _Rapid Maintenance_ or _RapMain_), it does not need any maintenance at all at least ideally. So I will not update the software often unless a bug is discovered.

As an user, as long as you compile the code with the latest version of Go1 compiler, it should work as good as new.

If it does not, fire an issue, I'll handle those ... from time to time.

Code will be signed with GPG key ID 37F8FB1ADE475E8D.

    -----BEGIN PGP PUBLIC KEY BLOCK-----

    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
    0XfypSirJA==
    =1CVS
    -----END PGP PUBLIC KEY BLOCK-----

## Credit

To many people who offered their direct great help during the making of this project on [StackOverflow](https://stackoverflow.com).

Thank you!

I don't even know you exist before [Google](https://www.google.com) brought me the answer that you posted 7 years ago.

The world will be a worse place without you! Thank you!
