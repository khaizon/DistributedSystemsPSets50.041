# DistributedSystemsPSets50.041

## **Assignment 2**

### Toh Kai Feng

### 1004581

## Introduction

This is the following file structure used:

![File Structure](images/filestructure.png)

1. To run questions P1 to P3, replace "--Question--" with either "P1_SharedPQ","P2_Voting","P3_LockServer" in the following command. (Note: p1_Measurement and p2_Measurement are additional scripts to measure the time of protocols):

```bash
go run -race --Question--/main.go
```

# Part 1

This would be the output when part 1 is ran:

![P1_output](images/p1_output.png)

what we are looking for is that the programme does not enter into the race condition. This can be seen with the print statements where each node is trying to add to a number with a single address. each node is trying to increment the value of the same number but must first acquire the lock with permission from all other nodes.

# Part 2

This would be the output when part 2 is ran:

![P2_output](images/p2_output.png)

Similarly, what we are looking for is that the programme does not enter into the race condition. This can be seen with the print statements where each node is trying to add to a number with a single address. each node is trying to increment the value of the same number but must first acquire the lock with permission from all other nodes. It is also interesting to note that voting protocol is much slower than lamport shared priority queue, presumably because more messages has to be exchanged before each node gets to enter the critical section. (second node has wait for first node to to release all votees and allow other nodes to vote before second node can enter critical section)

# Part 3

This would be the output when part 3 is ran:

![P3_output](images/p3_output.png)

What is different from the output for lockserver is that I am not printing any text about critical section because the lock server protocol is quite straight forward in its safety. (the lock server would queue all requests and allow the next node to enter only when earlier reqeusting nodes have completed)

# Performance Comparison

This is a graph plot comparing:

1. Lamport Shared Priority Queue (LP Shared)
2. Voting
3. Central Lock Server

![Comparison_output](images/comparison.png)

A best fit line is plotted to measure the rate of growth in time taken when the number of concurrent requests are made.

As mentioned, I believe voting protocol is much slower than lamport shared priority queue, presumably because more messages has to be exchanged before each node gets to enter the critical section. (second node has wait for first node to to release all votees and allow other nodes to vote before second node can enter critical section)
