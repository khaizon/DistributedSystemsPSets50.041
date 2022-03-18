# DistributedSystemsPSets50.041

## **Assignment 2**
### Toh Kai Feng
### 1004581
## Introduction
This is the following file structure used:

![File Structure](images/filestructure.png)

1. To run questions 1_1 to 1_3, replace "--Question--" with either "P1_1","P1_2","P1_3" in the following command:
```bash
go run -race PSet1/BroadcastingServer/--Question--/main.go 
```
2. To run questions 2_1 to 2_4, replace "--Question--" with either "P2_1","P2_2a", "P2_2b","P2_3", "P2_4" in the following command:
```bash
 go run -race PSet1/BullyAlgorithm/--Question--/main.go  
```

# Part 1 a
## Part 1
This would be the prompt when the program is ran:

![P1_1 input prompt](images/P1_1_input_prompt.png)

what we are looking for is that the programme does not enter into the race condition. This can be seen with the print statements where each node is trying to add to a number with a single address. each node is trying to increment the value of the same number but must first acquire the lock with permission from all other nodes.