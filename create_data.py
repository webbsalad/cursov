import os
import json
import random
import string

# Создаем папку data, если её нет
os.makedirs("data", exist_ok=True)

# Функция для генерации случайного JSON-контента
def generate_random_json():
    data = {
        "id": random.randint(1, 1000),
        "name": ''.join(random.choices(string.ascii_letters, k=10)),
        "value": random.random(),
        "description": ''.join(random.choices(string.ascii_letters + string.digits, k=20))
    }
    return data

# Генерация 100 JSON-файлов
for i in range(100000):
    filename = os.path.join("data", f"file_{i+1}.json")
    with open(filename, "w") as json_file:
        json.dump(generate_random_json(), json_file)

# Генерация большого текстового файла размером 100 МБ
filename = os.path.join("data", "large_text_file.txt")
with open(filename, "w") as txt_file:
    # Заполняем файл случайным текстом, пока его размер не достигнет 100 МБ
    while txt_file.tell() < 1 * 1024 * 1024 * 1024: 
        txt_file.write(''.join(random.choices(string.ascii_letters + string.digits + ' ', k=10024)) + "\n")

print("Файлы созданы в папке 'data'")
