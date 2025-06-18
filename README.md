# Banknote-Chain Source Code Instruction
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

## Step 4: Compile the system performance experiment project.
```
cd coinpurse
go build -o ./go_build_coinpurse_Linux_linux coinpursebench/bnbench.go
mv ./go_build_coinpurse_Linux_linux ../
cd ../
```


## Step 5: Run the system performance experiment.
```
./performance_experiment.sh
```

## Step 6: Draw the experiment figures.
Launch Jupyter Notebook, open drawpic.ipynb, and run it.