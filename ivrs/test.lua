local file_dir = "C:\\Users\\kolonse\\ivrs\\"
local menu_1 = file_dir .. "hello.wav"

engine:sleep(1000)
engine:playback(menu_1)

engine:log('info', 'IVR End')