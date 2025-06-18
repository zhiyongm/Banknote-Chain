import pandas as pd

csv_file = "./exp_data/tx_16_4920.csv"

chunksize = 10000000  

df = pd.DataFrame()


total_rows = 0
for chunk in pd.read_csv(csv_file, usecols=["banknote_hash", "bn_turnover_count"], chunksize=chunksize):
    
    chunk = chunk.dropna(subset=["bn_turnover_count"])

    
    df = pd.concat([df, chunk], ignore_index=True)

    
    total_rows += chunksize
    print(f"Processed {total_rows} rows, {total_rows / 1124408664 * 100:.4f}%")

df = df.sort_values(by="bn_turnover_count", ascending=False)

df = df.drop_duplicates(subset="banknote_hash", keep="first")


print(df.head())
df.to_csv("./exp_data/banknote_info.csv")