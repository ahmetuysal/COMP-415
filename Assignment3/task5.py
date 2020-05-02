from pyspark import SparkContext, SparkConf
from datetime import datetime
import matplotlib.pyplot as plt
import os

if __name__ == "__main__":
    conf = SparkConf().setAppName("compute sum").setMaster(
        "spark://ip-172-31-95-231.ec2.internal:7077")
    sc = SparkContext(conf=conf)

    # run it once to remove first time overhead
    sc.textFile("data/numbers.txt").flatMap(str.split).map(float).sum()

    # each results is a tuple of (sum, time it took to compute)
    results = []

    input_file_paths = ["data/numbers.txt", "data/numbers2.txt", "data/numbers4.txt",
                        "data/numbers8.txt", "data/numbers16.txt", "data/numbers32.txt"]

    for path in input_file_paths:
        start_time = datetime.now()
        lines = sc.textFile(path)
        result = lines.flatMap(str.split).map(float).sum()
        results.append(
            (result, (datetime.now() - start_time).total_seconds()))
    labels = list(map(lambda path: path[5:len(path) - 4], input_file_paths))

    file_sizes = list(
        map(lambda path: str(round(os.path.getsize(path) / (1024**2), 2)), input_file_paths))
    execution_times = list(map(lambda result: result[1], results))

    plt.ylabel("Execution Time")
    plt.xlabel("File size (MB)")
    plt.bar(file_sizes, execution_times)

    plt.savefig('execution_time.png')

    print(file_sizes)
    print(execution_times)
