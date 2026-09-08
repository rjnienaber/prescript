-- A port in miniature: it asks something, then prints numbers it drew at
-- random. Without a seed those numbers differ on every run, which is what
-- makes a program like this impossible to script.
io.write("HOW MANY? ")
io.output():setvbuf("no")
local count = tonumber(io.read("l"))
-- INT(RND(1) * 1000), the way a port translates it. Written the same way in
-- every language here, so a tape gives all of them the same numbers.
local drawn = {}
for _ = 1, count do
  drawn[#drawn + 1] = math.floor(math.random() * 1000)
end
print(table.concat(drawn, " "))
