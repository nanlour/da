# VDF-Based Nakamoto-Style Blockchains: A Proposal for Energy-Efficient Consensus

<div align="center">

**Student** Rong Wang  
**Student ID:** 1373126  
**Email:** rongwang@student.unimelb.edu.au

</div>

## Abstract

A purely peer-to-peer version of electronic cash would allow online payments to be sent directly from one party to another without going through a financial institution. While Bitcoin, the pioneering implementation of this concept, has demonstrated remarkable resilience and success through its Proof-of-Work (PoW) based Nakamoto consensus, its significant energy consumption has prompted exploration into alternative consensus mechanisms. Current Proof-of-Stake (PoS) systems, while more energy-efficient, often introduce complexity and may lack the straightforward, empirically demonstrated robustness of Nakamoto consensus. This paper proposes a novel Nakamoto-style consensus protocol that leverages Verifiable Delay Functions (VDFs) to simulate the probabilistic block generation of Bitcoin without the associated high energy expenditure, thereby offering a potentially more sustainable and scalable blockchain infrastructure.

## 1. Introduction

The core innovation of Bitcoin, the Nakamoto consensus, relies on PoW to secure the network and validate transactions. This mechanism, while effective, necessitates vast computational power, leading to concerns about environmental impact and operational costs. Many subsequent blockchain designs have adopted PoS mechanisms to mitigate these energy demands. However, the design of robust and secure PoS consensus can be intricate, and some implementations have faced challenges in clearly documenting their consensus rules or have required off-chain governance for dispute resolution.

Inspired by the properties of Verifiable Delay Functions (VDFs), which provide a proof of elapsed time that is slow to compute but fast to verify, we propose a new consensus mechanism. This mechanism aims to retain the fundamental principles of Nakamoto consensus—probabilistic block proposal, the longest chain rule for fork resolution, and incentive structures—while substituting the energy-intensive PoW with a VDF-based process proportional to a node's stake.

The design is guided by two primary principles:
* The fork with the most accumulated "work"—represented by validated blocks and, indirectly, by the stake committed to it—is considered the canonical chain. (The notion of "faster" block generation on such forks requires careful consideration, potentially relating to VDF parameter adjustments based on aggregate network conditions rather than individual fork strength to avoid unintended centralizing effects).
* A node's probability of proposing the next block and receiving the associated reward is directly proportional to its stake in the system.

Except for the mining and validation processes, the proposed system largely mirrors Bitcoin's architecture, including its transaction model and peer-to-peer network structure. A critical security measure is a single slashing condition: a node must not generate more than one block at any given height. This paper will focus on the novel aspects of this VDF-based design, assuming familiarity with the foundational elements of Bitcoin.

## 2. Epochs and Stake Management

To manage network parameters and stake dynamics, the system incorporates epochs.

* **Epoch Definition:** An epoch is a predefined, relatively long period (e.g., analogous to Solana's two-day epochs).
* **Epoch Functions:** Epochs serve two main purposes:
    1.  **Blockchain Finalization:** Providing a mechanism for achieving probabilistic finality, also we plan to use mechanism to achieve BFT's instant finality for each epoch.
    2.  **Stake Transitions:** Managing changes in stake distribution.
* **Mining Difficulty Seed:** For epoch $n$, the VDF mining difficulty is determined by a seed derived from the hash of the last block of epoch $n-2$. This delay (one epoch) is intended to prevent miners from strategically influencing or predicting the difficulty seed for their immediate advantage.
* **Stake Effective Date:** Stake transactions (e.g., bonding or unbonding stake) submitted during epoch $n$ only become effective at the commencement of epoch $n+2$. This latency ensures orderly stake adjustments and contributes to the stability of the consensus mechanism.

To ensure the mining difficulty seed is a publicly verifiable random number, yet prevent nodes from pre-calculating other nodes' specific VDF challenges far in advance, the following mechanism is proposed: The base seed is derived from the VDF output using the last block hash of epoch $n-2$. Each node then signs this base seed (or a derivative, for example based seed plus block height) with its private key. The hash of this signature can then be used to determine the node's specific VDF iteration target for block proposal. This signature ensures that a node's commitment to a particular challenge is verifiable upon block publication and discourages withholding blocks.

## 3. VDF-Based Mining and Block Generation

VDFs are integral to the proposed system in two key capacities:

1.  **Public Random Seed Generation:** As described above, VDFs are used to produce a trustworthy public random seed for determining mining challenges within an epoch. The VDF output, based on a historical block hash (from epoch $n-2$), is unpredictable at the time of that block's creation, preventing manipulation by the block's proposer to gain future mining advantages.
2.  **Proof of Elapsed Time (Mining):** VDFs act as a proof of "effort" or "time spent" working on a specific block, analogous to Bitcoin's PoW.

The mining process simulates Bitcoin's probabilistic block discovery. Bitcoin's block arrival times follow an exponential distribution. We can replicate this characteristic as follows:
1.  The probability density function (PDF) of an exponential distribution describes Bitcoin's block times.
2.  Integrating the PDF yields the cumulative distribution function (CDF), ranging from 0 to 1.
3.  A uniformly distributed random number $U \in [0,1]$ is generated (derivable from the node-specific VDF challenge process described earlier).
4.  This random number $U$ is input into the inverse of the CDF (the quantile function). The output of this quantile function will yield a value that follows the same exponential distribution as Bitcoin's block times.
5.  This output value is used as the time parameter (number of iterations) for the VDF computation.

Therefore, the block generation times in this system are designed to statistically mirror those of Bitcoin. If the network targets an average block time corresponding to $T$ VDF iterations, the conceptual probability $P$ of "solving" the VDF for a unit of work can be considered $1/T$. For a node $N$ with stake $s$ out of a total network stake $S$, its probability of proposing the next block is proportional to $s/S$. This is achieved by adjusting the VDF iteration count (time parameter) for node $N$ based on its stake. For instance, a node with a larger stake might be assigned a statistically shorter VDF computation time (a lower iteration count on average) compared to a node with a smaller stake, such that its probability of completing the VDF first aligns with its stake proportion. (The exact mapping is detailed in the project's [codebase](https://github.com/nanlour/da/blob/ee1aeaab654a4a33511fb3e87ca37c52be9d5d8c/src/ecdsa_da/crypto.go#L136-L160)).

## 4. Fork Choice Rule and Finality

* **Fork Choice Rule:** The system adheres to the longest chain principle, as in Bitcoin. The chain with the most accumulated valid blocks is considered the canonical chain.
* **Finality:** Finality is defined probabilistically based on epochs. When epoch $n+1$ commences, all blocks from epoch $n-1$ and earlier are considered finalized. This provides a stronger guarantee of immutability for older blocks than Bitcoin's probabilistic finality, which accrues with each new block but never reaches absolute certainty.

## 5. Slashing Conditions and Attack Mitigation

A primary security concern in stake-based systems is the "nothing-at-stake" problem. While VDFs introduce a cost (time), explicit penalties for malicious behavior are crucial.

* **Multiple Blocks at Same Height:** A node generating more than one block at a given height is a slashable offense. Unlike Bitcoin, where producing a block incurs significant energy cost, a VDF-based system might allow a malicious actor with sufficient computational resources (e.g., many cores for parallel VDF computations, if not designed carefully for a single actor at a single height) to attempt this. An automatic slashing mechanism will penalize nodes engaging in such behavior by confiscating a portion of their stake.

* **Long-Range Attack Mitigation:**
    A long-range attack involves an adversary creating an alternative chain from an early point in the blockchain's history, potentially to rewrite history or execute double-spends. Consider an attacker attempting to start an attack from epoch $n$. They might try to influence the last block of epoch $n$, denoted $B_n$, to gain an advantage in mining for epoch $n+2$, as the mining difficulty seed for epoch $n+2$ is derived from $B_n$.

    Let each epoch have an expected duration $T_{epoch}$. The VDF computation for generating the difficulty seed for epoch $n+2$ from $B_n$ is designed to take a significant amount of time, for example, $T_{VDF\_seed} \approx T_{epoch}/2$.

    An attacker aiming to manipulate the seed from $B_n$ must:
    1.  Finalize their choice of $B_n$ (which occurs by the end of epoch $n$ on the honest chain).
    2.  Wait for their VDF seed computation ($T_{VDF\_seed}$).
    3.  Begin building their alternative chain for epoch $n+2$ and beyond.

    According to the finality rule, blocks from epoch $n$ are finalized at the start of epoch $n+2$. The attacker, therefore, has a limited window to outpace the honest chain. The delay imposed by $T_{VDF\_seed}$ significantly curtails this window. If the legitimate chain finalizes blocks from epoch $n$ before the attacker's alternative chain gains traction and becomes longer, the attack fails. Given that an attacker would need to control a substantial portion of the stake (typically >50% in Nakamoto-style consensus for sustained attacks) and overcome the inherent time delays imposed by both block VDFs and the seed generation VDF, successfully mounting such a long-range attack is considered highly improbable, especially without the ability to predict VDF outputs far in advance.

## 6. Comparison with Other Blockchains

### 6.1. Bitcoin (Proof-of-Work)

* **Consensus:** Bitcoin uses PoW (SHA-256 hashing). The proposed system uses VDFs combined with stake.
* **Energy Consumption:** Bitcoin's PoW is extremely energy-intensive. The proposed VDF-based system is designed to be significantly more energy-efficient, as VDF computation, while time-consuming, does not require the massive parallel brute-force computation of PoW.
* **Block Proposers:** In Bitcoin, miners with higher hash power have a higher probability of finding a block. In the proposed system, nodes with higher stake have a higher probability.
* **Finality:** Bitcoin offers probabilistic finality (the "6-block rule" is a common heuristic). The proposed system introduces epoch-based finality, where blocks from epoch $n-1$ are considered final at the start of epoch $n+1$, offering a more deterministic finality horizon.
* **Hardware:** Bitcoin mining is dominated by ASICs. VDFs are designed to be ASIC-resistant, promoting more equitable participation, though specialized hardware for VDFs (if VDFs become widespread) could emerge. The key is that the "work" is sequential time, not parallel computation.

### 6.2. Ethereum (Proof-of-Stake)

* **Consensus:** Ethereum has transitioned to PoS (Casper-FFG and LMD GHOST). Ethereum's PoS involves validators locking up ETH, proposing, and attesting to blocks.
* **Energy Consumption:** Like other PoS systems, Ethereum's PoS is vastly more energy-efficient than PoW. The proposed VDF system shares this energy-efficiency benefit.
* **Complexity:** Ethereum's PoS, with its attestation committees, validator rotation, and intricate slashing conditions, is arguably more complex than the proposed system, which aims to more closely mimic the simpler probabilistic nature of Bitcoin's Nakamoto consensus.
* **Finality:** Ethereum PoS has notions of justified and finalized blocks, typically achieved within a couple of epochs (each epoch being 32 slots of 12 seconds, so around 12.8 minutes for justification and 25.6 minutes for finalization). The proposed system's epoch-based finality (potentially two days per epoch for finalization of $n-1$) is longer but conceptually simpler.
* **Slashing:** Both systems employ slashing for misbehavior. Ethereum has a broader range of slashable offenses. The proposed system currently focuses on a single, critical offense (double-signing at the same height).

### 6.3. Solana (Proof-of-History / Tower BFT)

* **Consensus:** Solana uses Proof-of-History (PoH), which is a VDF creating a verifiable sequence of events, combined with Tower BFT for consensus.
* **VDF Usage:** Solana's PoH is a core VDF that continuously runs, providing a global clock and timestamping transactions before they are batched into blocks. In the proposed system, VDFs are used discretely for block proposal "puzzles" and for random seed generation.
* **Block Production:** Solana has a leader schedule determined in advance, with leaders producing blocks for short slots. The proposed system uses a probabilistic, stake-weighted competition to produce blocks, similar to Bitcoin.
* **Documentation and Governance:** The initial motivation mentioned challenges with Solana's documentation and reliance on social consensus for penalizing bad behavior. The proposed system aims for clear, on-chain enforceable rules in the spirit of Nakamoto consensus.
* **Energy Consumption:** Solana is also energy efficient due to its PoS/PoH nature.

### 6.4. Other VDF-based Chains (e.g., Chia)

* **Consensus (Chia):** Chia uses Proof-of-Space-and-Time (PoST). Proof-of-Space involves allocating disk space, and Proof-of-Time is provided by VDFs.
* **Resource:** Chia's "work" is primarily tied to storage capacity ("plots") and then refined by VDFs. The proposed system's "work" is tied to stake and then proven by VDFs.
* **Similarities:** Both Chia and the proposed system use VDFs to introduce a verifiable time delay, preventing certain attacks and reducing energy use compared to PoW.
* **Differences:** The fundamental resource for participation differs (space vs. stake). The integration and specific roles of VDFs also vary according to each protocol's design.

### 6.5. Comparative Analysis

| Feature                  | Bitcoin PoW     | Proposed VDF-Stake        | Ethereum PoS              | Solana PoH                 | Chia PoST                   |
|--------------------------|-----------------|---------------------------|---------------------------|----------------------------|------------------------------|
| Consensus                | SHA-256 hashing | VDF + stake-weighting     | Casper-FFG + LMD GHOST    | Continuous PoH + Tower BFT | Proof-of-Space + VDF        |
| Energy Profile           | High            | Low                       | Low                       | Low                        | Low                         |
| Block Proposer Selection | Hash power      | Stake                     | Stake                     | Scheduled leader           | Plot & VDF                  |
| Fork Resolution          | Longest chain   | Longest chain             | GHOST & checkpoints       | BFT voting                 | Longest plot chain          |
| Finality                 | Probabilistic   | Epoch-based deterministic | Epoch-based probabilistic | Instant (BFT)              | Probabilistic (plots & VDF) |
| Slashing Complexity      | None            | Minimal (double-sign)     | Extensive (many offenses) | Social & protocol governed | Minimal (space proofs)      |

## 7. Advantages of the Proposed System

* **Energy Efficiency:** Drastically reduces energy consumption compared to PoW blockchains like Bitcoin.
* **Nakamoto-Style Security:** Aims to retain the simplicity and proven security model of Nakamoto consensus (longest chain rule, probabilistic block proposal proportional to committed resources).
* **Reduced Complexity (vs. some PoS):** Potentially offers a less complex consensus mechanism than some feature-rich PoS systems, making it easier to analyze and verify.
* **ASIC Resistance (VDF-dependent):** VDFs are generally designed to be hard to speed up significantly with parallel hardware, which could lead to more democratized participation if VDF hardware specialization remains limited.
* **Clear Finality:** Provides a clearer path to block finality through its epoch-based mechanism.

## 8. Current Work

The current project has implemented:
* A workable prototype of the proposed blockchain, without finalization and epoch change
* P2P network
* RPC server for client interaction
* Web UI for demo

## 9. Future Work

Future research should focus on:
* Formal security analysis of the proposed VDF-based consensus mechanism
* Performance optimization for VDF computation and verification
* Empirical evaluation of the system under various attack scenarios
* Investigation of additional slashing conditions to enhance security
* Exploration of VDF parameter tuning for optimal performance and security

## 10. Conclusion

This paper introduces a VDF-based Nakamoto-style blockchain designed to address the energy consumption issues of traditional PoW systems while striving for the conceptual elegance and proven resilience of Nakamoto consensus. By mapping stake to VDF challenge difficulty, the system allows for probabilistic block proposal proportional to a node's economic commitment, without requiring energy-intensive computations. The introduction of epochs facilitates stake management and provides a clear mechanism for block finality. While further research and development are needed, this VDF-based approach offers a promising direction for building secure, decentralized, and sustainable blockchain infrastructures.

## 11. References

1. Nakamoto, S. (2008). Bitcoin: A peer-to-peer electronic cash system.
2. Boneh, D., Bonneau, J., Bünz, B., & Fisch, B. (2018). Verifiable delay functions. In Annual International Cryptology Conference (pp. 757-788). Springer.
3. Buterin, V., & Griffith, V. (2017). Casper the friendly finality gadget.
4. Yakovenko, A. (2018). Solana: A new architecture for a high performance blockchain.
5. Cohen, B., & Pietrzak, K. (2019). The Chia network blockchain.
6. Garay, J., Kiayias, A., & Leonardos, N. (2015). The bitcoin backbone protocol: Analysis and applications.
7. Daian, P., Pass, R., & Shi, E. (2019). Snow White: Robustly reconfigurable consensus and applications to provably secure proof of stake.
8. Dwork, C., & Naor, M. (1992). Pricing via processing or combatting junk mail. In Annual International Cryptology Conference (pp. 139-147). Springer.
9. Gilad, Y., Hemo, R., Micali, S., Vlachos, G., & Zeldovich, N. (2017). Algorand: Scaling byzantine agreements for cryptocurrencies.
10. Buchman, E., Kwon, J., & Milosevic, Z. (2018). The latest gossip on BFT consensus.
11. Secure Randomness: From zero to verifiable delay Functions, Part 1. (2022, October 19). https://neodyme.io/en/blog/secure-randomness-part-1/#intro
12. Lopp, J. (n.d.). Bitcoin Block Time Variance: Theory vs Reality. Cypherpunk Cogitations. https://blog.lopp.net/bitcoin-block-time-variance/
