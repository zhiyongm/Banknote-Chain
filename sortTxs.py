import pandas as pd

file_name = 'token_transfers1y.csv'  
df = pd.read_csv(file_name)

token_value = '0xdac17f958d2ee523a2206206994597c13d831ec7' 
df_filtered = df[df['token_address'] == token_value]



df_filtered=df_filtered[df_filtered['value']!="0"]


df_sorted = df_filtered.sort_values(by='block_number', ascending=True)

output_file_name = file_name.replace('.csv', '_usdt_sorted.csv')

df_sorted.to_csv(output_file_name, index=False)

