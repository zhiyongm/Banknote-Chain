


import pandas as pd
import random
import string
import sys


ShardNumber = int(sys.argv[1])

BlockSize = 1000

RecycleInterval = 1000

RECYCLE_Length = int(sys.argv[2])

DefaultBalance = 100000000

CsvChunkSize = 10000

FlushBlockInterval = 100


InputCSV = sys.argv[5]


OutputTransactionsCSV = sys.argv[3]
OutputBlockstatsCSV = sys.argv[4]

ActiveBanknotes = {}

global_block_count = 0                  
global_processed_txs = 0                 
global_banknote_new_count = 0            
global_banknote_transfer_count = 0       
global_banknote_split_count = 0          
global_banknote_recycle_count = 0        
global_auto_recycle_count = 0            

active_banknote_count=0


current_block_tx_counter = 0





current_block_new = 0
current_block_transfer = 0
current_block_split = 0
current_block_recycle = 0






transactions_output_buffer = []


blockstats_output_buffer = []


account_modified_map = {}









def generate_random_hash(length=16):
    chars = string.ascii_letters + string.digits
    return ''.join(random.choices(chars, k=length))


def initialize_address_if_needed(address, current_block):
   
    global active_banknote_count
    if address not in ActiveBanknotes:
        
        
        
        

        
        ActiveBanknotes[address] = []
        for shard_id in range(ShardNumber):
            bn = {
                "denomination": DefaultBalance,
                "owner": address,
                "hash": generate_random_hash(),
                "shardnumber": shard_id,
                "lastseenblock": current_block,
                "turnovercount": 0,
                "creator": address
            }
            ActiveBanknotes[address].append(bn)
            active_banknote_count+=1
            
            
            

            
            
            
            
            
            
            
            
            


def get_banknotes_of_owner_in_all_shards(owner):

    if owner not in ActiveBanknotes:
        return []
    return ActiveBanknotes[owner]


def remove_banknote_from_owner(bn):    
    owner = bn["owner"]
    
    if owner in ActiveBanknotes:
        
        if len(ActiveBanknotes[owner]) == 0:
            del ActiveBanknotes[owner]
        try:
            ActiveBanknotes[owner].remove(bn)
            global active_banknote_count
            active_banknote_count-=1
        except:

            print("remove error!!!!!!!!!!!!!!!!",bn["hash"])
        
        
        


def add_banknote_to_owner(bn):
    owner = bn["owner"]
    if owner not in ActiveBanknotes:
        ActiveBanknotes[owner] = []
    ActiveBanknotes[owner].append(bn)

    global account_modified_map
    account_modified_map[owner] = True

    global active_banknote_count
    active_banknote_count+=1
    
    
    
    


def banknote_new(owner, creator,amount, current_block):
  
    global global_banknote_new_count
    global current_block_new

    shard_id = random.randint(0, ShardNumber-1 )
    
    

    
    bn = {
        "denomination": amount,
        "owner": owner,
        "hash": generate_random_hash(),
        "shardnumber": shard_id,
        "lastseenblock": current_block,
        "turnovercount": 0,
        "creator": creator
    }
    add_banknote_to_owner(bn)

    
    global_banknote_new_count += 1
    current_block_new += 1

    
    
    
    
    transactions_output_buffer.append({
        "sender":owner, "recipient":"", "amount":"",
        "BN_tx_type":1,
        "banknote_hash":bn["hash"],"banknote_value":amount,
        "shard_id":shard_id,"bn_turnover_count":bn["turnovercount"],"bn_creator":creator,
        "bn_new1_hash":"","newBN1_value":"",
        "bn_new2_hash":"","newBN2_value":"",
    })

    return bn


def banknote_transfer(bn, new_owner, current_block):
 
    global global_banknote_transfer_count
    global current_block_transfer

    old_owner = bn["owner"]
    bn_hash = bn["hash"]
    shard_id = bn["shardnumber"]
    amount = bn["denomination"]

    
    remove_banknote_from_owner(bn)
    
    bn["owner"] = new_owner
    
    bn["turnovercount"] += 1
    
    bn["lastseenblock"] = current_block

    
    add_banknote_to_owner(bn)

    
    global_banknote_transfer_count += 1
    current_block_transfer += 1

    
    
    
    
    transactions_output_buffer.append({
        "sender":"", "recipient":new_owner, "amount":"",
        "BN_tx_type":2,
        "banknote_hash":bn_hash,"banknote_value":amount,
        "shard_id":shard_id,"bn_turnover_count":bn["turnovercount"],"bn_creator":bn["creator"],
        "bn_new1_hash":"","newBN1_value":"",
        "bn_new2_hash":"","newBN2_value":"",
    })


def banknote_recycle(bn, current_block):
  
    global global_banknote_recycle_count
    global current_block_recycle
    global global_auto_recycle_count

    owner = bn["owner"]
    shard_id = bn["shardnumber"]
    amount = bn["denomination"]
    bn_hash = bn["hash"]

    
    

    
    remove_banknote_from_owner(bn)

    global_banknote_recycle_count += 1
    current_block_recycle += 1
    global_auto_recycle_count += 1  

    
    
    
    

    transactions_output_buffer.append({
        "sender":owner, "recipient":owner, "amount":"",
        "BN_tx_type":4,
        "banknote_hash":bn_hash,"banknote_value":amount,
        "shard_id":shard_id,"bn_turnover_count":bn["turnovercount"],"bn_creator":bn["creator"],
        "bn_new1_hash":"","newBN1_value":"",
        "bn_new2_hash":"","newBN2_value":"",
    })


def banknote_split(bn, A, B, amountA, amountB, current_block):
  
    global global_banknote_split_count
    global current_block_split

    old_owner = bn["owner"]
    shard_id = bn["shardnumber"]
    old_hash = bn["hash"]
    old_turnover = bn["turnovercount"]

    
    remove_banknote_from_owner(bn)

    
    bn1 = {
        "denomination": amountA,
        "owner": A,
        "hash": generate_random_hash(),
        "shardnumber": shard_id,
        "lastseenblock": current_block,
        "turnovercount": old_turnover,
        "creator": bn["creator"]
    }
    add_banknote_to_owner(bn1)

    
    bn2 = {
        "denomination": amountB,
        "owner": B,
        "hash": generate_random_hash(),
        "shardnumber": shard_id,
        "lastseenblock": current_block,
        "turnovercount": old_turnover,
        "creator": bn["creator"]
    }
    add_banknote_to_owner(bn2)

    
    global_banknote_split_count += 1
    current_block_split += 1

    
    
    
    
    
    
    transactions_output_buffer.append({
        "sender":old_owner, "recipient":old_owner, "amount":"",
        "BN_tx_type":3,
        "banknote_hash":old_hash,"banknote_value":bn["denomination"],
        "shard_id":shard_id,"bn_turnover_count":bn["turnovercount"],"bn_creator":bn["creator"],
        "bn_new1_hash":bn1["hash"],"newBN1_value":bn1["denomination"],
        "bn_new2_hash":bn2["hash"],"newBN2_value":bn2["denomination"],
    })

def process_transfer(A, B, x, current_block):
  
    

    initialize_address_if_needed(A, current_block)
    initialize_address_if_needed(B, current_block)

    
    A_banknotes = get_banknotes_of_owner_in_all_shards(A)
    
    for bn in A_banknotes:
        if bn["denomination"] == x:
            
            banknote_transfer(bn, B, current_block)
            return


    if account_modified_map.get(A):
        
        
        del account_modified_map[A]
        
    else:
        
        
        pass
    

    
    accum = 0

    for bn in A_banknotes:
        if accum >= x:
            break

        d = bn["denomination"]
        if accum + d < x:
            
            banknote_transfer(bn, B, current_block)
            accum += d
        elif accum + d == x:
            
            banknote_transfer(bn, B, current_block)
            accum += d
            break
        else:
            
            
            over = (accum + d) - x
            
            amountToB = x - accum
            amountToA = over  

            banknote_split(bn, A, B, amountToA, amountToB, current_block)
            accum += amountToB
            break

    if accum < x:
        
        needed = x - accum
        
        banknote_new(B, A,needed, current_block)
        
        







def finalize_block():
  
    global global_block_count
    global current_block_tx_counter
    global current_block_new
    global current_block_transfer
    global current_block_split
    global current_block_recycle

    global global_processed_txs
    global global_banknote_new_count
    global global_banknote_transfer_count
    global global_banknote_split_count
    global global_banknote_recycle_count

    

    
    if global_block_count % RecycleInterval == 0:

        
        for owner, bn_list in ActiveBanknotes.items():
            for bn in bn_list:
                if (global_block_count - bn["lastseenblock"]) > RECYCLE_Length:
                    
                    banknote_recycle(bn, global_block_count)
        


    
    active_banknotes_count = active_banknote_count
    blockstats_output_buffer.append([
        global_block_count,
        active_banknotes_count,
        global_banknote_new_count,
        global_banknote_transfer_count,
        global_banknote_split_count,
        global_banknote_recycle_count,
        global_processed_txs,
        current_block_new,
        current_block_transfer,
        current_block_split,
        current_block_recycle
    ])

    
    
    
    
    
    




    
    if global_block_count % FlushBlockInterval == 0:
        flush_to_csv()

    
    current_block_tx_counter = 0
    current_block_new = 0
    current_block_transfer = 0
    current_block_split = 0
    current_block_recycle = 0


    global_block_count += 1

def flush_to_csv():

    global transactions_output_buffer
    
    global blockstats_output_buffer

    
    if len(transactions_output_buffer) > 0:
        
        df_tx = pd.DataFrame(transactions_output_buffer, columns=[
            "sender", "recipient", "amount",
            "BN_tx_type",
            "banknote_hash","banknote_value", "shard_id","bn_turnover_count","bn_creator",
            "bn_new1_hash","newBN1_value",
            "bn_new2_hash","newBN2_value",

        ])
        
        
    
        write_header = False
        try:
            with open(OutputTransactionsCSV, 'r'):
                pass
        except FileNotFoundError:
            write_header = True

        df_tx.to_csv(OutputTransactionsCSV, mode='a', index=False, header=write_header)

        
        transactions_output_buffer = []

    
    
    
    
    
    
    
    
    
    
    
    
    
    
    
    
    
    
    
    


    
    if len(blockstats_output_buffer) > 0:
        df_bs = pd.DataFrame(blockstats_output_buffer, columns=[
            "block_number",
            "active_banknotes_count",
            "total_new",
            "total_transfer",
            "total_split",
            "total_recycle",
            "processed_raw_txs",
            "block_new",
            "block_transfer",
            "block_split",
            "block_recycle"
        ])

        write_header = False
        try:
            with open(OutputBlockstatsCSV, 'r'):
                pass
        except FileNotFoundError:
            write_header = True

        df_bs.to_csv(OutputBlockstatsCSV, mode='a', index=False, header=write_header)

        
        blockstats_output_buffer = []





import os
from tqdm import tqdm
def main():

    pbar = tqdm(total=59872552)
    file_path=OutputTransactionsCSV
    try:
        os.remove(file_path)
        print(f"文件 {file_path} 已成功删除")
    except FileNotFoundError:
        print(f"文件 {file_path} 不存在")
    except PermissionError:
        print(f"没有权限删除文件 {file_path}")
    except Exception as e:
        print(f"删除文件时发生错误: {e}")

    file_path=OutputBlockstatsCSV
    try:
        os.remove(file_path)
        print(f"文件 {file_path} 已成功删除")
    except FileNotFoundError:
        print(f"文件 {file_path} 不存在")
    except PermissionError:
        print(f"没有权限删除文件 {file_path}")
    except Exception as e:
        print(f"删除文件时发生错误: {e}")

    global current_block_tx_counter
    global global_processed_txs
    
    

    csv_iter = pd.read_csv(InputCSV, usecols=["from_address", "to_address", "value"],
                           chunksize=CsvChunkSize)

    for df_chunk in csv_iter:
        
        for idx, row in df_chunk.iterrows():
            A = str(row["from_address"])
            B = str(row["to_address"])
            x = int(row["value"])


            transactions_output_buffer.append({
                "sender":A, "recipient":B, "amount":x,
                "BN_tx_type":0,
            })
            
            process_transfer(A, B, x, global_block_count)
            transactions_output_buffer.append({
                "sender":"~"
            })
            
            global_processed_txs += 1
            pbar.update(1)
            
            current_block_tx_counter += 1

            
            if current_block_tx_counter >= BlockSize:
                finalize_block()



    
    if current_block_tx_counter > 0:
        finalize_block()

    
    flush_to_csv()


if __name__ == "__main__":
    
    try:
        main()
    except KeyboardInterrupt:
        print("程序被中断")
        sys.exit(0)