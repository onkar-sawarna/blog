---
title: "The API should see MySQL, not the topology"
description: "The shop grew more API boxes, then split into two services. Each one used to know the write server and the read server. A proxy answered as MySQL and kept that list itself."
pubDate: 2026-09-11
tags: ["systems"]
---

A buyer taps Buy. A thousand other people open the same pair of shoes, item 42. I add more API boxes so the shop can take the load.

Each box already keeps a few MySQL connections open and reuses them. I wrote that story when [a thousand clicks met four sockets](/blog/request-hedging-is-a-second-get-not-a-bigger-pool/). I thought more boxes would just mean more of those small pools, and the database would be fine.

It was not fine. The database saw every box. And a dummy network hop in front of MySQL would not have fixed it. The hop had to read the SQL.

## Too many doors into MySQL

I ran two hundred API processes. Each one kept ten connections to MySQL. If they all woke up at once, that is two thousand clients.

MySQL has a limit, `max_connections`. Mine was 400. After 400 open clients, the next one is refused. The shop starts seeing "too many connections."

Those two thousand clients were not two thousand different questions. Most of them wanted item 42. Each process still opened its own connections, and each new connection still pays to start TCP: three packets to open (SYN, SYN-ACK, ACK) and four to close (FIN and ACK both ways). I paid that cost from every process.

<figure>
  <img src="/blog/proxysql-before.svg" alt="Two hundred API processes each hold a pool of ten MySQL connections. MySQL has max_connections of 400 and is out of clients." width="720" height="300" />
  <figcaption>Figure 1. Each process brought its own pool. MySQL saw all of them.</figcaption>
</figure>

A load balancer that only forwards TCP would not have saved me. It never looks at the query. It cannot send a read to one machine and a write to another. It only sees ports and bytes.

## The API still thinks it is talking to MySQL

I put ProxySQL in the middle.

That is the useful part for the shop. I can send API requests to the right database machine without changing the Buy code, the shoe page, or any other business logic. Catalog still says `SELECT` for item 42. Checkout still says `INSERT` for `o1`. I did not add "if this is a read, go to the replica" in the application. The proxy reads the SQL and picks the server.

The API code did not change how it talks to the database. Same MySQL driver. Same user and password. Same kind of connection. ProxySQL answers the MySQL hello. As far as the API can tell, it connected to MySQL.

The host in the config is no longer the real database. It is the proxy. The buyer still hits the API. The API still runs a `SELECT` for the shoe, or an `INSERT` for order `o1`. Those queries stop at ProxySQL first.

ProxySQL keeps its own set of connections already open to the real MySQL. That is a pool again, but one pool in front of the database, shared by every API box.

MySQL now sees forty clients, not two thousand. Those forty were opened when the proxy started. A new API box does not open forty more.

<figure>
  <img src="/blog/proxysql-path.svg" alt="Buyer to API to ProxySQL to MySQL. MySQL sees forty clients." width="720" height="280" />
  <figcaption>Figure 2. The API talks to the proxy. The database only sees the connections the proxy already opened.</figcaption>
</figure>

This is different from the pool inside one API process. That pool lives on one machine. The next machine cannot use it. The proxy pool sits in front of MySQL, so every API box can use it.

<figure>
  <img src="/blog/proxysql-looks-like-mysql.svg" alt="The API connects to host port 3306 with a MySQL user and password. ProxySQL answers the MySQL handshake. The real MySQL servers sit behind it." width="720" height="260" />
  <figcaption>Figure 3. One address in the app. The MySQL protocol is only on this hop. The list of real servers stays behind the proxy.</figcaption>
</figure>

## Who knows which machine is which

Topology here just means the map: which machine takes writes (the primary), which machines take reads (the replicas).

There are two places that map can live.

**The API knows the map.** Writes go to `primary:3306`. Reads go to `replica:3306`. That is fine when the shop is one program. Every new service then copies those two addresses. When the primary dies and another machine becomes primary, every service must change, or every service must notice the change itself. I have gotten that notice wrong.

**The API knows one address.** `proxy:3306`. It looks like MySQL because it talks like MySQL. ProxySQL holds the map. It knows the primary, the replicas, and which SQL should go where. When the primary changes, I update the proxy. I do not change application code, and I do not redeploy catalog and checkout.

<figure>
  <img src="/blog/proxysql-topology.svg" alt="Left: the API has a write host and a read host. Right: the API has one proxy host. The proxy holds primary and replicas." width="720" height="300" />
  <figcaption>Figure 4. Same SQL. The question is who owns the map of servers.</figcaption>
</figure>

## Splitting the shop into two services

The shop used to be one program. The shoe page and Buy lived together, so the two database addresses lived in one place.

Then I split it. Catalog serves item 42. Checkout writes order `o1`.

I did not put routing in those services. I defined routing rules on the proxy: this `SELECT` goes to the replica group, this `INSERT` goes to the primary. That is the whole move. Catalog and checkout still talk to one MySQL host. They still run the same queries they ran in the monolith. The business logic did not change. The proxy already knew where each query should land.

If each service owned the map instead, I would have copied the primary and the replica into two codebases. After a failover, catalog might still point at a dead machine. Checkout might write to a machine that is only a replica now. Rules on the proxy avoid that. I can go from one program to two services as soon as the rules exist.

<figure>
  <img src="/blog/proxysql-split.svg" alt="A monolith splits into catalog and checkout. Both connect to ProxySQL, which looks like MySQL. One MySQL cluster sits behind it." width="720" height="300" />
  <figcaption>Figure 5. Only the routing rules changed. Catalog and checkout still see one MySQL host. It was the proxy.</figcaption>
</figure>

## One MySQL connection, many waiting APIs

An API worker often keeps a connection open after it finishes item 42, ready for the next request. If that connection is a real MySQL connection, it still counts against the 400 limit, even while it sits idle.

ProxySQL does not have to hold a real MySQL connection for an idle API. It reads the query, borrows one of its MySQL connections, waits for the answer, sends the answer back, and can lend that same MySQL connection to someone else. Many API sessions share fewer MySQL connections. People call that multiplex. It only means: do not save a database connection for a client that is not asking a question right now.

One MySQL connection still runs one query at a time. Sharing does not make one query faster.

<figure>
  <img src="/blog/proxysql-mux.svg" alt="Many API client sessions enter ProxySQL. ProxySQL reads each query and picks a free backend socket. MySQL has two busy sockets and a small live set." width="720" height="300" />
  <figcaption>Figure 6. An idle API session does not each own a MySQL connection.</figcaption>
</figure>

A transaction is different. `BEGIN` means the next statements must run on the same server, in the same session, so they see their own writes. ProxySQL then sticks that API to one MySQL connection until `COMMIT` or `ROLLBACK`. Buy often does this. If I start a transaction and then wait on Redis, I am holding a MySQL connection for no query. The forty connections fill up and I blame the proxy.

## The shoe is a read. Buy is a write.

Item 42 is a read. Buy is a write. Those should not hit the same machine if I have a primary and a replica.

A TCP-only proxy cannot tell them apart. ProxySQL can, because it reads the SQL. A `SELECT` for the shoe can go to a replica. An `INSERT` for `o1` can go to the primary.

The lists it chooses from are just named groups of servers. People call a group a hostgroup. One group is the primary. Another is the replicas. A rule says: this kind of SQL goes to that group.

<figure>
  <img src="/blog/proxysql-route.svg" alt="API to ProxySQL. SELECT item 42 goes to a replica. INSERT order o1 goes to the primary." width="720" height="300" />
  <figcaption>Figure 7. A TCP load balancer cannot make this split. It never sees the query.</figcaption>
</figure>

A replica can be late. The buyer pays, then refreshes, and the read goes to a replica that has not got `o1` yet. The confirmation looks empty. For a read that must see the write I just did, I still send it to the primary. The proxy will not guess that for me.

## More than one proxy

If I have only one ProxySQL box, and it dies, the shop thinks MySQL is down. The shop thought that box was MySQL.

So I run more than one. The API still has one name. A DNS name or a small balancer picks proxy a, b, or c. Each one answers like MySQL. Each one has its own pool to the real database.

That is the trap. Three proxies with forty connections each can mean 120 clients on MySQL, not 40. I added copies so the shop survives a dead proxy. I also used up more of the 400 limit. The database cap does not grow because I added proxies.

The copies must agree on the map. If proxy a writes to the new primary and proxy b still writes to the old one, catalog and checkout disagree again. I scale the proxy so one box can die. I still pick one number for how many MySQL connections I am allowed, and I split that number across the copies.

They also do not gossip. A routing rule lives on that one process. If I change "send this `SELECT` to the primary" on proxy a, proxy b and proxy c keep the old rule. The API still has one name, so the next Buy might land on a or on c. Half the shop follows the new path. Half does not. I apply the same rule change on every proxy, or I have not changed the rule.

<figure>
  <img src="/blog/proxysql-scale.svg" alt="The API can use ProxySQL a, b, or c. Each holds forty backend connections. MySQL sees up to one hundred and twenty clients." width="720" height="300" />
  <figcaption>Figure 8. Scale the proxy. Keep one budget on the database. Three pools of 40 is not still 40.</figcaption>
</figure>

<figure>
  <img src="/blog/proxysql-no-gossip.svg" alt="ProxySQL a has a new rule sending SELECT to the primary. ProxySQL b and c still send SELECT to the replica. They do not share the change." width="720" height="260" />
  <figcaption>Figure 9. They do not tell each other. A rule change is a change on every box.</figcaption>
</figure>

## What I took from this

A pool inside one API process saves that process from opening MySQL on every click. It does not save me when I have two hundred processes.

ProxySQL looks like MySQL to the app. The app keeps one address. I define routing rules on the proxy. That is enough to cut a monolith into services. Catalog and checkout do not learn the map, and I do not change their business logic. A read goes to a replica and a write goes to the primary because a rule said so.

MySQL should see a limited number of clients, not one connection per API worker. Reads and writes can go to different machines because the proxy reads the SQL, not because I hid two IPs behind one port. When I add proxy boxes, MySQL sees the sum of their pools. A new routing rule is not one write. It is the same write on every proxy, because the copies do not tell each other.

I still get this wrong. I size the proxy pool to "how many API workers I have." I leave a transaction open while I call Redis. I send every `SELECT` to a replica and then cannot find the order on the confirmation page. I treat the proxy like a dumb load balancer on the MySQL port. I add three proxies and hit `max_connections` anyway. I change a rule on one box and debug the other two for an hour.

If this is useful, wrong, or incomplete, write to me.
