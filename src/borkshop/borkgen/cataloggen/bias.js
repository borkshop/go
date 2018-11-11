
module.exports = Bias;

function byFrequency(frequency) {
    return function compareFrequency(a, b) {
        return frequency[b] - frequency[a];
    };
}

function Bias(frequency) {
    this.frequency = frequency;
    this.keys = Object.keys(frequency);
    this.keys.sort(byFrequency(frequency));
    this.cumulative = [];
    this.total = 0;
    for (let i = 0; i < this.keys.length; i++) {
        let key = this.keys[i];
        this.total += frequency[key];
        this.cumulative.push(this.total);
    }
}

Bias.prototype.pick = function pick() {
    let r = Math.random() * this.total;
    for (let i = 0; i < this.cumulative.length; i++) {
        if (r <= this.cumulative[i]) {
            return this.keys[i];
        }
    }
    throw new Error('assertion ' + r + ' ' + i + ' ' + this.total + ' ' + util.inspect(this));
};

Bias.table = biasTable;

function biasTable(frequencyTable) {
    let biases = {};
    let letters = Object.keys(frequencyTable);
    for (let i = 0; i < letters.length; i++) {
        let letter = letters[i];
        biases[letter] = new Bias(frequencyTable[letter]);
    }
    return biases;
}