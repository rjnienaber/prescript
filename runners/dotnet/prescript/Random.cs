// Makes a .NET port's randomness repeatable, in one of two ways.
//
// With PRESCRIPT_TAPE set, every draw returns a value recorded from the
// reference BASIC interpreter, in order, so this port and every other port of
// the same program draw the same numbers. Without it the draws come from a
// seeded generator of this file's own, which makes this port repeatable but
// leaves it drawing a different sequence from most other languages. See
// docs/determinism.md.
//
// Activated by compiling this file into the port, which prescript.targets does
// without the port being edited. Declaring System.Random in source shadows the
// runtime's: the C# compiler prefers a type it can see the source of, so every
// `new Random()`, `Random.Shared` and `rnd.Next()` in the port binds here
// instead of to System.Private.CoreLib.
//
// The other shims replace a function at run time. .NET has no seam for that:
// `new Random()` takes entropy in a constructor with no global seed to set,
// and reaching its methods afterwards means detouring compiled code, which
// needs a bytecode library that has to be taught about every new runtime. The
// compiler is the seam .NET does have, and it needs nothing but source.
//
// What this does not reach: RandomNumberGenerator and the rest of
// System.Security.Cryptography, Guid.NewGuid, and anything a NuGet dependency
// compiled for itself before it got here.

// Nullable annotations are a property of the port, not of this file, and this
// file is compiled into ports that have made either choice. Turning them off
// here keeps it from warning in one kind of project or the other.
#nullable disable

namespace System
{
    /// <summary>
    /// System.Random's surface, answered from one place.
    ///
    /// Everything is virtual where the real one is virtual, so a port that
    /// subclasses Random still compiles and still overrides what it meant to.
    /// The seed a constructor is handed is ignored on purpose: a port that
    /// seeds itself is repeatable already, and what is wanted here is for it
    /// to be repeatable in step with every other port of the same program.
    /// </summary>
    public class Random
    {
        public static Random Shared { get; } = new Random();

        public Random()
        {
        }

        // Named as the runtime names it, capital and all, so a port calling
        // new Random(Seed: 42) still compiles.
        public Random(int Seed)
        {
        }

        public virtual int Next() => (int)(global::Prescript.Tape.Next() * int.MaxValue);

        // What a BASIC port writes for INT(RND(1) * N), and what the other
        // shims do with the same value, so one tape means one number.
        public virtual int Next(int maxValue) => (int)(global::Prescript.Tape.Next() * maxValue);

        public virtual int Next(int minValue, int maxValue) =>
            minValue + (int)(global::Prescript.Tape.Next() * ((long)maxValue - minValue));

        public virtual double NextDouble() => global::Prescript.Tape.Next();

        public virtual float NextSingle() => (float)global::Prescript.Tape.Next();

        public virtual long NextInt64() => (long)(global::Prescript.Tape.Next() * long.MaxValue);

        public virtual long NextInt64(long maxValue) => (long)(global::Prescript.Tape.Next() * maxValue);

        public virtual long NextInt64(long minValue, long maxValue) =>
            minValue + (long)(global::Prescript.Tape.Next() * ((double)maxValue - minValue));

        public virtual void NextBytes(byte[] buffer) => NextBytes(new Span<byte>(buffer));

        // One draw per byte, which is what makes a count of draws comparable
        // between a port that asks for bytes and one that asks for numbers.
        public virtual void NextBytes(Span<byte> buffer)
        {
            for (int index = 0; index < buffer.Length; index++)
            {
                buffer[index] = (byte)(global::Prescript.Tape.Next() * 256);
            }
        }

        protected virtual double Sample() => global::Prescript.Tape.Next();

        public T[] GetItems<T>(T[] choices, int length) => GetItems(new ReadOnlySpan<T>(choices), length);

        public T[] GetItems<T>(ReadOnlySpan<T> choices, int length)
        {
            T[] chosen = new T[length];
            GetItems(choices, new Span<T>(chosen));
            return chosen;
        }

        public void GetItems<T>(ReadOnlySpan<T> choices, Span<T> destination)
        {
            for (int index = 0; index < destination.Length; index++)
            {
                destination[index] = choices[Next(choices.Length)];
            }
        }

        public void Shuffle<T>(T[] values) => Shuffle(new Span<T>(values));

        // The same Fisher-Yates the runtime's own Shuffle does, drawing the
        // same number of times, so a port that shuffles stays in step with one
        // that picks indexes by hand.
        public void Shuffle<T>(Span<T> values)
        {
            for (int index = 0; index < values.Length - 1; index++)
            {
                int swap = Next(index, values.Length);
                (values[index], values[swap]) = (values[swap], values[index]);
            }
        }
    }
}

namespace Prescript
{
    using System;
    using System.IO;

    /// <summary>
    /// Where the numbers come from.
    ///
    /// A port built from several projects compiles this file once per
    /// assembly, and separate copies drawing separately would put the same
    /// program out of step with itself. The drawing function is therefore
    /// created once and parked on the AppDomain, where every copy finds it:
    /// the copies are different types, but Func&lt;double&gt; is the runtime's
    /// own, so they can all hold the same one.
    /// </summary>
    internal static class Tape
    {
        private const string Key = "prescript.random.draw";

        private static readonly Func<double> Draw = Install();

        public static double Next() => Draw();

        private static Func<double> Install()
        {
            // The AppDomain is the one object every copy of this file
            // already agrees on, which is exactly what makes it the thing to
            // lock while deciding who creates the shared function.
            lock (AppDomain.CurrentDomain)
            {
                if (AppDomain.CurrentDomain.GetData(Key) is Func<double> shared)
                {
                    return shared;
                }

                Func<double> draw = Create();
                AppDomain.CurrentDomain.SetData(Key, draw);
                return draw;
            }
        }

        private static Func<double> Create()
        {
            string path = Environment.GetEnvironmentVariable("PRESCRIPT_TAPE");
            if (string.IsNullOrEmpty(path))
            {
                return Seeded(Environment.GetEnvironmentVariable("PRESCRIPT_SEED"));
            }

            double[] values = Read(path);
            Action<int> record = Usage(Environment.GetEnvironmentVariable("PRESCRIPT_TAPE_USAGE"));
            int drawn = 0;

            return () =>
            {
                // Running off the end is an error rather than a wrap. A port
                // that draws more than the reference did has restructured how
                // it consumes randomness, which is a finding; silently
                // starting the tape again would turn that finding into a
                // transcript that merely looks wrong somewhere further on.
                //
                // Reported and exited rather than thrown, because a thrown
                // error arrives as a stack trace and the line that says what
                // went wrong scrolls off the top of the failure report.
                if (drawn >= values.Length)
                {
                    Console.Error.WriteLine("prescript: the tape ran out after " + values.Length
                        + " values; this port draws more randomness than the reference did");
                    Environment.Exit(1);
                }

                double value = values[drawn++];
                record(drawn);
                return value;
            };
        }

        private static double[] Read(string path)
        {
            try
            {
                var values = new System.Collections.Generic.List<double>();
                foreach (string line in File.ReadLines(path))
                {
                    string trimmed = line.Trim();
                    if (trimmed.Length == 0 || trimmed[0] == '#')
                    {
                        continue;
                    }

                    values.Add(double.Parse(trimmed, global::System.Globalization.CultureInfo.InvariantCulture));
                }

                return values.ToArray();
            }
            catch (Exception failure)
            {
                Console.Error.WriteLine("prescript: could not read the tape " + path + ": " + failure.Message);
                Environment.Exit(1);
                return null;
            }
        }

        /// <summary>
        /// How much of the tape a port used is the number that separates a
        /// real logic bug from a port that restructured its draws, so it is
        /// written down rather than left to be inferred from the transcript.
        ///
        /// Rewritten after every draw rather than once on exit, because the
        /// number that matters is how much had been drawn when the run stopped
        /// agreeing with the transcript -- and a run that diverged by hanging
        /// is killed, which runs no exit handler at all. Rewriting at offset 0
        /// is safe without truncating: the count only ever grows, so its
        /// decimal form only ever gets longer and each write covers the last.
        /// </summary>
        private static Action<int> Usage(string path)
        {
            if (string.IsNullOrEmpty(path))
            {
                return drawn => { };
            }

            FileStream file;
            try
            {
                file = new FileStream(path, FileMode.Create, FileAccess.Write, FileShare.ReadWrite);
            }
            catch (Exception)
            {
                // A report that cannot say how much was drawn is worth less
                // than one that can, and worth far more than a run killed for
                // failing to say it.
                return drawn => { };
            }

            Action<int> record = drawn =>
            {
                try
                {
                    byte[] count = global::System.Text.Encoding.UTF8.GetBytes(drawn + "\n");
                    file.Seek(0, SeekOrigin.Begin);
                    file.Write(count, 0, count.Length);
                    file.Flush(true);
                }
                catch (Exception)
                {
                }
            };

            // Written from the start, so that a port which drew nothing at all
            // says so rather than saying nothing.
            record(0);
            return record;
        }

        /// <summary>
        /// mulberry32, the same generator the Node shim uses, and for the same
        /// reason: short enough to read in one sitting, and its whole state is
        /// a single 32-bit word, which is what makes the sequence identical
        /// everywhere. Two shims sharing an algorithm is not a coincidence to
        /// hide -- seeded .NET and seeded Node draw the same numbers, which is
        /// as much as a seed can offer and less than a tape does.
        ///
        /// The runtime's own seeded Random is unreachable from here: this file
        /// is what System.Random means in the compilation it is part of.
        /// </summary>
        private static Func<double> Seeded(string seed)
        {
            uint state = unchecked((uint)(int.TryParse(seed, out int parsed) ? parsed : 0) + 0x9e3779b9u);

            return () =>
            {
                unchecked
                {
                    state += 0x6d2b79f5u;
                    uint t = state;
                    t = (t ^ (t >> 15)) * (t | 1u);
                    t ^= t + (t ^ (t >> 7)) * (t | 61u);
                    return ((t ^ (t >> 14)) & uint.MaxValue) / 4294967296.0;
                }
            };
        }
    }
}
