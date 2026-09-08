package prescript;

import java.io.BufferedReader;
import java.io.FileReader;
import java.io.RandomAccessFile;
import java.util.Random;
import java.util.random.RandomGenerator;

/**
 * Where a Java port's randomness comes from once the agent has redirected it,
 * in one of two ways.
 *
 * <p>With PRESCRIPT_TAPE set, every draw returns a value recorded from the
 * reference BASIC interpreter, in order, so this port and every other port of
 * the same program draw the same numbers. Without it, one seeded
 * {@link Random} answers every draw, which makes this port repeatable but
 * leaves it drawing a different sequence from every other language. See
 * docs/determinism.md.
 *
 * <p>Every method takes the generator the port called, and ignores it. That is
 * the point: two Random instances in one program, and Math.random's own, all
 * draw from here, so the order of the draws is the order the program makes
 * them rather than an accident of how many generators it built.
 *
 * <p>Lives in the agent jar, which the JVM appends to the system class path
 * when it loads the agent, so rewritten port code can reach it by name.
 */
public final class Tape {
    private static final double[] VALUES = readTape(System.getenv("PRESCRIPT_TAPE"));
    private static final Random SEEDED = VALUES == null ? seededGenerator() : null;
    private static final RandomAccessFile USAGE = openUsage(System.getenv("PRESCRIPT_TAPE_USAGE"));

    private static int drawn;

    private Tape() {
    }

    /** Math.random(). */
    public static double random() {
        return next();
    }

    public static double nextDouble(RandomGenerator generator) {
        return next();
    }

    public static float nextFloat(RandomGenerator generator) {
        return (float) next();
    }

    public static boolean nextBoolean(RandomGenerator generator) {
        return next() < 0.5;
    }

    public static int nextInt(RandomGenerator generator, int bound) {
        // What a BASIC port writes for INT(RND(1) * N), and what the other
        // shims do with the same value, so one tape means one number.
        return (int) (next() * bound);
    }

    public static int nextInt(RandomGenerator generator, int origin, int bound) {
        return origin + (int) (next() * (bound - origin));
    }

    public static int nextInt(RandomGenerator generator) {
        // Spread over the whole range, which is what nextInt() promises. Ports
        // of these games do not use it; it is here so that one that does is
        // still repeatable rather than silently drawing from somewhere else.
        return (int) (next() * 4294967296.0 + Integer.MIN_VALUE);
    }

    public static long nextLong(RandomGenerator generator) {
        return (long) ((next() - 0.5) * 2.0 * Long.MAX_VALUE);
    }

    private static synchronized double next() {
        if (VALUES == null) {
            drawn++;
            recordUsage();
            return SEEDED.nextDouble();
        }

        // Running off the end is an error rather than a wrap. A port that
        // draws more than the reference did has restructured how it consumes
        // randomness, which is a finding; silently starting the tape again
        // would turn that finding into a transcript that merely looks wrong
        // somewhere further on.
        //
        // Reported and exited rather than thrown, because a thrown error
        // arrives as a stack trace and the line that says what went wrong
        // scrolls off the top of the failure report.
        if (drawn >= VALUES.length) {
            System.err.println("prescript: the tape ran out after " + VALUES.length
                    + " values; this port draws more randomness than the reference did");
            System.exit(1);
        }

        double value = VALUES[drawn++];
        recordUsage();
        return value;
    }

    private static Random seededGenerator() {
        String seed = System.getenv("PRESCRIPT_SEED");
        return new Random(seed == null || seed.isEmpty() ? 0L : Long.parseLong(seed.trim()));
    }

    private static double[] readTape(String path) {
        if (path == null || path.isEmpty()) {
            return null;
        }

        try (BufferedReader reader = new BufferedReader(new FileReader(path))) {
            double[] values = new double[64];
            int count = 0;
            for (String line = reader.readLine(); line != null; line = reader.readLine()) {
                line = line.strip();
                if (line.isEmpty() || line.startsWith("#")) {
                    continue;
                }
                if (count == values.length) {
                    double[] grown = new double[count * 2];
                    System.arraycopy(values, 0, grown, 0, count);
                    values = grown;
                }
                values[count++] = Double.parseDouble(line);
            }

            double[] exact = new double[count];
            System.arraycopy(values, 0, exact, 0, count);
            return exact;
        } catch (Exception failure) {
            System.err.println("prescript: could not read the tape " + path + ": " + failure);
            System.exit(1);
            return null;
        }
    }

    /**
     * How much of the tape a port used is the number that separates a real
     * logic bug from a port that restructured its draws, so it is written down
     * rather than left to be inferred from the transcript. Written from the
     * start, so that a port which drew nothing at all says so rather than
     * saying nothing.
     */
    private static RandomAccessFile openUsage(String path) {
        if (path == null || path.isEmpty()) {
            return null;
        }

        try {
            RandomAccessFile file = new RandomAccessFile(path, "rws");
            file.setLength(0);
            file.write("0\n".getBytes());
            return file;
        } catch (Exception failure) {
            return null;
        }
    }

    /**
     * Rewritten after every draw rather than once on exit, because the number
     * that matters is how much had been drawn when the run stopped agreeing
     * with the transcript -- and a run that diverged by hanging is killed,
     * which runs no shutdown hook at all. Rewriting at offset 0 is safe
     * without truncating: the count only ever grows, so its decimal form only
     * ever gets longer and each write covers the last.
     */
    private static void recordUsage() {
        if (USAGE == null) {
            return;
        }

        try {
            USAGE.seek(0);
            USAGE.write((drawn + "\n").getBytes());
        } catch (Exception ignored) {
            // A report that cannot say how much was drawn is worth less than
            // one that can, and worth far more than a run killed for failing
            // to say it.
        }
    }
}
