let fs = require('fs');
let util = require('util');
let Bias = require('./bias');

function main() {
    let corpus = fs.readFileSync('corpus.txt', 'utf8') 
    let lines = corpus.trim().split(/\n/);
    let words = new Map();
    for (let i = 0; i < lines.length; i++) {
        let line = lines[i];
        let parts = line.split(/\s+/);
        var word = parts[0];
        var frequency = +parts[1];
        words.set(word, frequency);
    }

    var lengthBias = new Bias(lengthFrequencies(words));
    let l2r = new Set(quantity(suffixNames(words), 100000));
    let r2l = new Set(quantity(prefixNames(words), 100000));
    let sample = [...l2r].filter((word) => r2l.has(word));
    sample = sample.filter((name) => !words.has(name));
    report(sampleByLengthBias(sample, lengthBias, 1200));
}

function prefixFrequencies(words) {
    let frequencies = {};
    for (let [word, f] of words.entries()) {
        let pword = '  ' + word + '  ';
        for (let j = 0; j < word.length + 1; j++) {
            let prefix = pword.slice(j, j + 2);
            let suffix = pword.slice(j + 2, j + 3);
            frequencies[prefix] = frequencies[prefix] || {};
            frequencies[prefix][suffix] = frequencies[prefix][suffix] || 0;
            frequencies[prefix][suffix] += f;
        }
    }
    return frequencies;
}

function suffixFrequencies(words) {
    let frequencies = {};
    for (let [word, f] of words.entries()) {
        let pword = '  ' + word + '  ';
        for (let j = 0; j < word.length + 2; j++) {
            let prefix = pword.slice(j, j + 1);
            let suffix = pword.slice(j + 1, j + 3);
            frequencies[suffix] = frequencies[suffix] || {};
            frequencies[suffix][prefix] = frequencies[suffix][prefix] || 0;
            frequencies[suffix][prefix] += f
        }
    }
    return frequencies;
}

function prefixName(table) {
    let name = '';
    let prev = '  ';
    for (;;) {
        let bias = table[prev];
        let next = bias.pick();
        if (next === " ") {
            return name;
        }
        name += next;
        prev = ("  " + name).slice(-2);
    }
}

function suffixName(table) {
    let name = '';
    let prev = '  ';
    for (;;) {
        let bias = table[prev];
        let next = bias.pick();
        if (next === " ") {
            return name;
        }
        name = next + name;
        prev = (name + "  ").slice(0, 2);
    }
}

function *suffixNames(words) {
    let table = Bias.table(prefixFrequencies(words));
    for (;;) {
        yield prefixName(table);
    }
}

function *prefixNames(words) {
    let frequencies = suffixFrequencies(words)
    let table = Bias.table(frequencies);
    for (;;) {
        yield suffixName(table);
    }
}

function quantity(iterable, quantity) {
    let set = new Set();
    for (let name of iterable) {
        if (set.size >= quantity) {
            return set;
        }
        set.add(name);
    }
}

function groupByLength(words) {
    let lengths = {};
    for (let i = 0; i < words.length; i++) {
        let word = words[i];
        lengths[word.length] = lengths[word.length] || [];
        lengths[word.length].push(word);
    }
    return lengths;
}

function pluck(array) {
    let index = Math.floor(Math.random() * array.length);
    let value = array[index]
    let last = array[array.length - 1];
    array[array.length - 1] = value;
    array[index] = last;
    return array.pop();
}

function sampleByLengthBias(words, lengthBias, quantity) {
    let result = new Set();
    let groups = groupByLength(words);
    for (let i = 0; i < quantity; i++) {
        let length = lengthBias.pick();
        let sample = groups[length];
        if (sample != null && sample.length > 0) {
            let name = pluck(sample);
            result.add(name);
        }
    }
    return result;
}

function lengthFrequencies(words) {
    var lengths = {};
    for (let [word, f] of words.entries()) {
        lengths[word.length] = lengths[word.length] || 0;
        lengths[word.length] += f;
    }
    return lengths;
}

function report(names) {
    for (let name of names) {
        console.log(name.toUpperCase());
    }
}

main();
