// A port in miniature: it asks something, then prints numbers it drew at
// random. Without a seed those numbers differ on every run, which is what makes
// a program like this impossible to script.
//
// Run by dotnet run rng.cs, the file-based app the .NET 10 SDK compiles and
// runs in one step, which is the analogue of go run and java Rng.java.

Console.Write("HOW MANY? ");
int count = int.Parse(Console.ReadLine()!.Trim());

var random = new Random();
var drawn = new string[count];
for (int index = 0; index < count; index++)
{
    // INT(RND(1) * 1000), the way a port translates it. Written the same way
    // in every language here, so a tape gives all of them the same numbers.
    drawn[index] = ((int)(random.NextDouble() * 1000)).ToString();
}

Console.WriteLine(string.Join(" ", drawn));
