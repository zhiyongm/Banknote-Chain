# Banknote-Chain Source Code Instruction

> 2025-09-21 Revision Version

## Step 1: Export the USDT transaction dataset for block numbers 18910000-21526000 on the Ethereum mainnet.

```
pip install ethereum-etl matplotlib numpy pandas scipy 
ethereumetl export_token_transfers --start-block 18910000 --end-block 21526000 --provider-uri http://ethereum-rpc-url --tokens 0xdAC17F958D2ee523a2206206994597C13D831ec7 --output token_transfers1y.csv
```

## Step 2: Sort the USDT dataset in ascending order of block numbers and filter out contract calls with a value of 0.

```
python sortTxs.py
```

## Step 3: Run the banknote circulation experiment.

```
python tx_simulator.py 16 164 tx_16_164.csv status_16_164.csv token_transfers1y_usdt_sorted.csv
python tx_simulator.py 16 1148 tx_16_1148.csv status_16_1148.csv token_transfers1y_usdt_sorted.csv
python tx_simulator.py 16 4920 tx_16_4920.csv status_16_4920.csv token_transfers1y_usdt_sorted.csv
python get_banknote_info.py
```

## Step 4: Compile the Banknote-Chain Prototype System.
```
make all
```
Then select the compiled product binary of the your CPU architecture and operating system version.

## Step 5: Edit configs as your server environment.
Modify all configuration json files in the expConfigs folder according to your server information, modify "storage_root_path, input_dataset_path, output_dataset_path, peer_count (the number of nodes except the leader node), proposer_address, proposer_address, p2p_sender_listen_address, usdt_dataset_path, banknote_dataset_path".


## Step 6: Run distributed performance experiment.
1. Copy the ``expConfigs`` folder , ``publisher_key`` the executale binary file to each experiment server.
2. Check whether the p2p ports of all servers are allowed in the firewall. If not, please allow them first.
3. Run `` blockcooker-yourCPUarch-yourOS -c EXP_PAPAMATER_leader.json`` on the bootstrap server.
4. Run `` blockcooker-yourCPUarch-yourOS -c EXP_PAPAMATER_peer.json`` on the  peer servers.
5. After completing a round of experiments, continue to execute all experiments in the configuration folder in the same way.

## Step 7: Run system scalability experiment.
1. Run `` blockcooker-yourCPUarch-yourOS -c scalability_1w_PLEDGERNUMBER_core.json`` in sequence.


## Step 6: Draw the experiment figures.
Launch at ``expDataAndFigsCode`` folder, open ``draw_banknote_circulation.ipynb`` and ``,draw_performance.ipynb``, edit the path of output experiment dataset, and run them.

We also present the experiment data of the paper at ``expDataAndFigsCode`` folder