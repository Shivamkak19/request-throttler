# Request throttler

Consider the following problem statement. 

In this problem, an ingestion infrastructure (called `connectors`) is based on a stateful queue system. This system handles tasks that typically involve pulling data from external platforms and upserting documents to the associated connected data source. These tasks are designed to be idempotent, allowing them to be retried with minimal side effects. Task execution is managed concurrently by workers running in our infrastructure on a service called `connectors-worker`, which operates 4 Kubernetes pods.

# Problem

Running activities, we often run into rate-limits which triggers retryable failures. We try to manage the concurrency for each user and platform but it’s by construction imperfect (in face of varying rate-limits conditions) and it prevents us from defining different priority levels for different workflows operating on the same connection.

*As an example you may want to have a high priority workflow in charge of fetching most recent data and a low priority workflow in charge of garbage collecting old data.*

Design a request throttling system to optimize requests rate based on the rate-limits we have on each platform for each user which is capable of scheduling requests based on their priorities.

# Formalization and assumptions

## Definitions

- **Platform**: an external API we make requests to using their client libraries.
- **Workspace**: a workspace connected to a set of platforms.
- **Connection**: the identity and credentials of a workspace on a platform.
- **Client**: the client library used to query a platform given a connection.
- **Request**: a request to a platform using a client for a given connection.
- **Workflow**: A set of activities being executed to achieve a goal.
- **Activity**: an idempotent piece of code that involves requests.
- **Priority**: the priority of a workflow.
- **Worker**: A machine (a k8s pod in our case) in charge of executing activities.
- **Rate Limit**: a fixed number of allowed request per rolling time window.

## Assumptions

- At any point of time multiple workflows (with respective priority) can be running concurrently for a given workspace and platform (hence connection). Among these workflows activities can be run concurrently as well within each workflow.
- Activities are scheduled and executed on `w>1` workers. For us `w=4` today but it can be arbitrarily high.
- We will assume that rate-limits are known and fixed for all platforms. When we hit rate limits some platform return up to date rate limitation information. We will not take these into account and solely rely on fixed rate limit values.

## Invariants

- Requests associated with activities of higher priority are always executed before requests associated with activities of lower priority.
- For a given priority level, requests are executed FIFO.

# Goal

Design and implement a request throttler in Typescript or Rust based on the formalization and assumptions above that enforce the invariants proposed. 

```tsx
interface RateLimit {
  requestCount: number;    // Number of requests allowed per rolling window.
  windowSeconds: number;   // Time window defined in seconds.
};

interface Connection {
  platform: string;      // A unique platform identifier.
  connection: string;    // A unique connection identifier.
  niceness: number;      // Associated workflow priority, 0 is the highest priority.
  rateLimit: RateLimit;  // The rate-limit for the connection.
};

export function throttle<T, U>(
  connection: Connection,
  fn: (arg: T) => Promise<U>,
);

```

Example usage:

```tsx
const activity = async (connection: Connection, ...) => {
  // ...
  client = PlatformClient.create(connection);
  const res = throttle(connection, async () => {
    await client.request();
  });
  // ...
};
```