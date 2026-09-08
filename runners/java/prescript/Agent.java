package prescript;

import java.lang.classfile.ClassFile;
import java.lang.classfile.ClassTransform;
import java.lang.classfile.CodeTransform;
import java.lang.classfile.instruction.InvokeInstruction;
import java.lang.constant.ClassDesc;
import java.lang.constant.MethodTypeDesc;
import java.lang.instrument.ClassFileTransformer;
import java.lang.instrument.Instrumentation;
import java.security.ProtectionDomain;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.atomic.AtomicBoolean;

/**
 * Makes a Java port's randomness repeatable without editing the port.
 *
 * <p>Activated by -javaagent:&lt;this jar&gt;, which the JVM loads before the
 * program's main method, so the program does not have to know it exists.
 *
 * <p>Unlike the Ruby, Python and Node shims, this one cannot simply replace a
 * function: Java resolves java.util.Random's methods at the call site and
 * there is no global to reassign. So it rewrites the call sites instead. Every
 * class the program loads is read on its way in, and each call to Math.random
 * or to one of Random's drawing methods is re-pointed at {@link Tape}, which
 * answers from the tape or from one seeded generator.
 *
 * <p>Rewriting the program rather than the JDK is the deliberate choice.
 * Redefining java.util.Random itself would reach further -- Collections.shuffle
 * and the random streams draw through it -- but it means retransforming a core
 * class that is loaded and compiled before any agent runs. The program's own
 * classes are loaded after this transformer is installed, so they are rewritten
 * on the way in, which needs no retransformation and cannot leave the JDK in a
 * state it was not built for.
 *
 * <p>The rewrite is done with java.lang.classfile, the class-file API the JDK
 * has shipped since 24. That is why this runner asks for a JDK that new: the
 * alternative is a third-party bytecode library, and a pinned environment is
 * worth more than support for an older JVM that nothing here has to run on.
 *
 * <p>What this does not reach: SecureRandom, SplittableRandom, a generator
 * built by RandomGeneratorFactory, and any draw made inside the JDK on the
 * program's behalf -- Collections.shuffle(list, rnd), rnd.ints(), and the rest
 * of the stream methods.
 */
public final class Agent {
    private static final ClassDesc TAPE = ClassDesc.of("prescript.Tape");

    /**
     * The receiver becomes the first argument and is then ignored. It has to
     * be taken because the call site leaves it on the stack, and ignoring it
     * is what makes every generator in a program draw from one place.
     */
    private static final String GENERATOR = "Ljava/util/random/RandomGenerator;";

    /**
     * Owners whose drawing methods are redirected. Random covers a port that
     * holds a Random, RandomGenerator one that holds the interface, and
     * ThreadLocalRandom the modern way of asking for the same thing.
     *
     * <p>SecureRandom is not here on purpose. It extends Random, so it would
     * cost one line -- but a program that asks for cryptographic randomness
     * has asked for something a recorded tape is not, and quietly answering it
     * from a text file is not a decision to take on a program's behalf.
     */
    private static final String[] OWNERS = {
        "java/util/Random",
        "java/util/random/RandomGenerator",
        "java/util/concurrent/ThreadLocalRandom",
    };

    /** Drawing methods, by descriptor, and what each becomes. */
    private static final String[][] DRAWS = {
        {"nextDouble", "()D"},
        {"nextFloat", "()F"},
        {"nextBoolean", "()Z"},
        {"nextInt", "()I"},
        {"nextInt", "(I)I"},
        {"nextInt", "(II)I"},
        {"nextLong", "()J"},
    };

    private static final Map<String, Redirect> REDIRECTS = redirects();

    private Agent() {
    }

    public static void premain(String arguments, Instrumentation instrumentation) {
        instrumentation.addTransformer(new Redirector());
    }

    private record Redirect(String name, MethodTypeDesc descriptor) {
    }

    private static Map<String, Redirect> redirects() {
        Map<String, Redirect> redirects = new HashMap<>();
        redirects.put("java/lang/Math.random()D",
                new Redirect("random", MethodTypeDesc.ofDescriptor("()D")));

        for (String owner : OWNERS) {
            for (String[] draw : DRAWS) {
                String name = draw[0];
                String descriptor = draw[1];
                redirects.put(owner + "." + name + descriptor,
                        new Redirect(name, MethodTypeDesc.ofDescriptor("(" + GENERATOR + descriptor.substring(1))));
            }
        }
        return redirects;
    }

    private static final class Redirector implements ClassFileTransformer {
        /**
         * Whether this class belongs to the runtime rather than to the program
         * under test.
         *
         * <p>The bootstrap loader is the obvious half. The other half is less
         * obvious and matters more: java rng.java compiles the source in this
         * same JVM, so javac's own classes -- loaded by the platform loader,
         * which is not null -- come past here too, and rewriting a compiler's
         * draws would spend the tape before the program had started.
         */
        private static boolean platform(Module module, ClassLoader loader) {
            if (loader == null) {
                return true;
            }
            if (module == null || !module.isNamed()) {
                return false;
            }

            String moduleName = module.getName();
            return moduleName.equals("java") || moduleName.startsWith("java.")
                    || moduleName.equals("jdk") || moduleName.startsWith("jdk.");
        }

        @Override
        public byte[] transform(Module module, ClassLoader loader, String name,
                Class<?> beingRedefined, ProtectionDomain domain, byte[] original) {
            if (name == null || name.startsWith("prescript/") || platform(module, loader)) {
                return null;
            }

            try {
                AtomicBoolean rewrote = new AtomicBoolean();
                CodeTransform redirect = (builder, element) -> {
                    if (element instanceof InvokeInstruction invoke) {
                        Redirect target = REDIRECTS.get(invoke.owner().asInternalName()
                                + "." + invoke.name().stringValue() + invoke.type().stringValue());
                        if (target != null) {
                            builder.invokestatic(TAPE, target.name(), target.descriptor());
                            rewrote.set(true);
                            return;
                        }
                    }
                    builder.with(element);
                };

                ClassFile classFile = ClassFile.of();
                byte[] rewritten = classFile.transformClass(classFile.parse(original),
                        ClassTransform.transformingMethodBodies(redirect));

                // Null means "left alone", which is both faster and safer than
                // handing back a re-encoded copy of a class nothing was going
                // to change.
                return rewrote.get() ? rewritten : null;
            } catch (Throwable failure) {
                // Said out loud rather than swallowed. A transformer that
                // throws is ignored by the JVM, and a port that quietly went
                // back to drawing its own numbers is exactly the silent
                // failure this whole exercise exists to prevent.
                System.err.println("prescript: could not redirect randomness in " + name + ": " + failure);
                return null;
            }
        }
    }
}
