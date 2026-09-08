// A port in miniature: it asks something, then prints numbers it drew at
// random. Without a seed those numbers differ on every run, which is what makes
// a program like this impossible to script.
//
// Run by the single-file source launcher, java rng.java, which is the Java
// analogue of go run: no build step, no class file left behind.

import java.io.BufferedReader;
import java.io.IOException;
import java.io.InputStreamReader;
import java.util.Random;

public class Rng {
    public static void main(String[] arguments) throws IOException {
        System.out.print("HOW MANY? ");
        System.out.flush();

        BufferedReader input = new BufferedReader(new InputStreamReader(System.in));
        int count = Integer.parseInt(input.readLine().strip());

        Random random = new Random();
        StringBuilder drawn = new StringBuilder();
        for (int index = 0; index < count; index++) {
            if (index > 0) {
                drawn.append(' ');
            }
            // INT(RND(1) * 1000), the way a port translates it. Written the
            // same way in every language here, so a tape gives all of them the
            // same numbers.
            drawn.append((int) (random.nextDouble() * 1000));
        }
        System.out.println(drawn);
    }
}
