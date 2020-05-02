from pyspark import SparkContext, SparkConf

if __name__ == "__main__":
    conf = SparkConf().setAppName("character count").setMaster(
        "spark://ip-172-31-95-231.ec2.internal:7077")
    sc = SparkContext(conf=conf)

    lines = sc.textFile("data/book.txt")

    characters = lines.flatMap(list)

    character_counts = characters.countByValue()

    for character, count in character_counts.items():
        print("{}: {}, ".format(character, count), end="")
